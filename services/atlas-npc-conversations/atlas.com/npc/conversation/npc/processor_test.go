package npc

import (
	"atlas-npc-conversations/test"
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func countTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return te
}

func insertCountRow(t *testing.T, p Processor, npcId uint32) {
	t.Helper()
	m := createTestModel(t, npcId)
	if _, err := p.Create(m); err != nil {
		t.Fatalf("Create npc conversation %d: %v", npcId, err)
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
	insertCountRow(t, p, 1000)
	insertCountRow(t, p, 1001)

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

	insertCountRow(t, p1, 2000)
	insertCountRow(t, p1, 2001)
	insertCountRow(t, p2, 3000)
	insertCountRow(t, p2, 3001)
	insertCountRow(t, p2, 3002)

	count, _, err := p1.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2 for tenant 1, got %d", count)
	}
}
