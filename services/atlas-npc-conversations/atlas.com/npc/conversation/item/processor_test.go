package item

import (
	"atlas-npc-conversations/test"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	logtest "github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func countTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return te
}

func createTestModel(t *testing.T, itemId uint32) Model {
	t.Helper()
	state := buildFixtureState(t, "intro")
	m, err := NewBuilder().
		SetItemId(itemId).
		SetNpcId(2084002).
		SetScriptName("compassUse").
		SetStartState("intro").
		AddState(state).
		Build()
	if err != nil {
		t.Fatalf("building test model: %v", err)
	}
	return m
}

func insertCountRow(t *testing.T, p Processor, itemId uint32) {
	t.Helper()
	m := createTestModel(t, itemId)
	if _, err := p.Create(m); err != nil {
		t.Fatalf("Create item conversation %d: %v", itemId, err)
	}
}

func TestProcessorImpl_Count_Empty(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	p := NewProcessor(l, ctx, db)
	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
	if updated != nil {
		t.Errorf("Expected nil updatedAt, got %v", updated)
	}
}

func TestProcessorImpl_Count_Populated(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	p := NewProcessor(l, ctx, db)
	insertCountRow(t, p, 2430001)
	insertCountRow(t, p, 2430002)

	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
	if updated == nil {
		t.Fatalf("updatedAt is nil; expected non-nil")
	}
	if time.Since(*updated) > 5*time.Second {
		t.Errorf("updatedAt too old: %v", *updated)
	}
}

func TestProcessorImpl_Count_TenantIsolation(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	te1 := countTestTenant(t)
	te2 := countTestTenant(t)
	ctx1 := tenant.WithContext(context.Background(), te1)
	ctx2 := tenant.WithContext(context.Background(), te2)
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	p1 := NewProcessor(l, ctx1, db)
	p2 := NewProcessor(l, ctx2, db)

	insertCountRow(t, p1, 2430003)
	insertCountRow(t, p1, 2430004)
	insertCountRow(t, p2, 2430005)
	insertCountRow(t, p2, 2430006)
	insertCountRow(t, p2, 2430007)

	count, _, err := p1.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2 for tenant 1, got %d", count)
	}
}

// TestProcessorImpl_ByItemIdProvider_Lookup verifies ByItemIdProvider — the
// internal lookup Task 8 (item-use consumer) drives; it resolves by item id,
// not by conversation id.
func TestProcessorImpl_ByItemIdProvider_Lookup(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	p := NewProcessor(l, ctx, db)
	insertCountRow(t, p, 2430008)

	m, err := p.ByItemIdProvider(2430008)()
	if err != nil {
		t.Fatalf("ByItemIdProvider(2430008): %v", err)
	}
	if m.ItemId() != 2430008 {
		t.Errorf("got itemId %d want 2430008", m.ItemId())
	}

	if _, err := p.ByItemIdProvider(9999999)(); err == nil {
		t.Error("expected error for unknown itemId")
	}
}

// TestProcessorImpl_CRUD exercises Create/Update/Delete/ByIdProvider end to end.
func TestProcessorImpl_CRUD(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	p := NewProcessor(l, ctx, db)

	created, err := p.Create(createTestModel(t, 2430009))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Id() == uuid.Nil {
		t.Fatal("Create did not assign an id")
	}

	fetched, err := p.ByIdProvider(created.Id())()
	if err != nil {
		t.Fatalf("ByIdProvider: %v", err)
	}
	if fetched.ItemId() != 2430009 {
		t.Errorf("fetched itemId: got %d want 2430009", fetched.ItemId())
	}

	updateModel := createTestModel(t, 2430009)
	updated, err := p.Update(created.Id(), updateModel)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Id() != created.Id() {
		t.Errorf("Update changed id: got %s want %s", updated.Id(), created.Id())
	}

	if err := p.Delete(created.Id()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := p.ByIdProvider(created.Id())(); err == nil {
		t.Error("expected error after delete")
	}
}

// TestProcessorImpl_DeleteAllForTenant verifies tenant-scoped bulk delete.
func TestProcessorImpl_DeleteAllForTenant(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	p := NewProcessor(l, ctx, db)
	insertCountRow(t, p, 2430010)
	insertCountRow(t, p, 2430011)

	n, err := p.DeleteAllForTenant()
	if err != nil {
		t.Fatalf("DeleteAllForTenant: %v", err)
	}
	if n != 2 {
		t.Errorf("DeleteAllForTenant: got %d want 2", n)
	}

	count, _, err := p.Count()
	if err != nil {
		t.Fatalf("Count() after delete: %v", err)
	}
	if count != 0 {
		t.Errorf("Count() after delete: got %d want 0", count)
	}
}
