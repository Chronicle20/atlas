package redis

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func newTestTenant(t *testing.T, region string) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), region, 0, 83)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func TestTenantRegistry_Clear_EmptyNamespace(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer client.Close()
	reg := NewTenantRegistry[string, string](client, "test:clear", func(k string) string { return k })
	tm := newTestTenant(t, "GMS")

	deleted, err := reg.Clear(context.Background(), tm)
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
}

func TestTenantRegistry_Clear_DeletesAllForTenant(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer client.Close()
	reg := NewTenantRegistry[string, string](client, "test:clear", func(k string) string { return k })
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if err := reg.Put(ctx, tm, fmt.Sprintf("k%d", i), "v"); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	deleted, err := reg.Clear(ctx, tm)
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if deleted != 5 {
		t.Fatalf("deleted = %d, want 5", deleted)
	}
	for i := 0; i < 5; i++ {
		ok, _ := reg.Exists(ctx, tm, fmt.Sprintf("k%d", i))
		if ok {
			t.Fatalf("key k%d still exists after Clear", i)
		}
	}
}

func TestTenantRegistry_Clear_TenantIsolation(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer client.Close()
	reg := NewTenantRegistry[string, string](client, "test:clear", func(k string) string { return k })
	tA := newTestTenant(t, "GMS")
	tB := newTestTenant(t, "GMS")
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_ = reg.Put(ctx, tA, fmt.Sprintf("k%d", i), "vA")
		_ = reg.Put(ctx, tB, fmt.Sprintf("k%d", i), "vB")
	}

	deleted, err := reg.Clear(ctx, tA)
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("deleted = %d, want 3", deleted)
	}
	for i := 0; i < 3; i++ {
		if ok, _ := reg.Exists(ctx, tA, fmt.Sprintf("k%d", i)); ok {
			t.Fatalf("tenant A key k%d should be gone", i)
		}
		if ok, _ := reg.Exists(ctx, tB, fmt.Sprintf("k%d", i)); !ok {
			t.Fatalf("tenant B key k%d should still exist", i)
		}
	}
}

func TestTenantRegistry_Clear_NamespaceIsolation(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer client.Close()
	regA := NewTenantRegistry[string, string](client, "test:clear:A", func(k string) string { return k })
	regB := NewTenantRegistry[string, string](client, "test:clear:B", func(k string) string { return k })
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		_ = regA.Put(ctx, tm, fmt.Sprintf("k%d", i), "vA")
		_ = regB.Put(ctx, tm, fmt.Sprintf("k%d", i), "vB")
	}

	deleted, err := regA.Clear(ctx, tm)
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if deleted != 4 {
		t.Fatalf("deleted = %d, want 4", deleted)
	}
	for i := 0; i < 4; i++ {
		if ok, _ := regA.Exists(ctx, tm, fmt.Sprintf("k%d", i)); ok {
			t.Fatalf("regA key k%d should be gone", i)
		}
		if ok, _ := regB.Exists(ctx, tm, fmt.Sprintf("k%d", i)); !ok {
			t.Fatalf("regB key k%d should still exist", i)
		}
	}
}

func TestTenantRegistry_Clear_RaceCleanWithPut(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer client.Close()
	reg := NewTenantRegistry[string, string](client, "test:clear:race", func(k string) string { return k })
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	for i := 0; i < 50; i++ {
		_ = reg.Put(ctx, tm, fmt.Sprintf("seed%d", i), "v")
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			i := 0
			for {
				select {
				case <-stop:
					return
				default:
					_ = reg.Put(ctx, tm, fmt.Sprintf("w%d-k%d", w, i), "v")
					i++
				}
			}
		}(w)
	}

	if _, err := reg.Clear(ctx, tm); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	close(stop)
	wg.Wait()
}

func TestTenantRegistry_GetAllEntries(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer client.Close()
	reg := NewTenantRegistry[string, string](client, "test:entries", func(k string) string { return k })
	t1 := newTestTenant(t, "GMS")
	t2 := newTestTenant(t, "EMS")
	ctx := context.Background()

	_ = reg.Put(ctx, t1, "100", "v100")
	_ = reg.Put(ctx, t1, "200", "v200")
	_ = reg.Put(ctx, t2, "300", "v300")

	entries, err := reg.GetAllEntries(ctx, t1)
	if err != nil {
		t.Fatalf("GetAllEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries for t1, got %d: %v", len(entries), entries)
	}
	if entries["100"] != "v100" {
		t.Fatalf("expected entries[\"100\"] = \"v100\", got %q", entries["100"])
	}
	if entries["200"] != "v200" {
		t.Fatalf("expected entries[\"200\"] = \"v200\", got %q", entries["200"])
	}
	// t2's key must not appear.
	if _, ok := entries["300"]; ok {
		t.Fatal("t2 key \"300\" should not appear in t1 GetAllEntries")
	}
}

func TestTenantRegistry_GetAllEntries_SkipsInternalKeys(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer client.Close()
	reg := NewTenantRegistry[string, string](client, "test:entries:internal", func(k string) string { return k })
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	_ = reg.Put(ctx, tm, "real", "val")
	// Directly insert an internal key to simulate internal state.
	internalKey := tenantEntityKey("test:entries:internal", tm, "_expiry")
	_ = reg.Client().Set(ctx, internalKey, `"internal"`, 0).Err()

	entries, err := reg.GetAllEntries(ctx, tm)
	if err != nil {
		t.Fatalf("GetAllEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (internal key skipped), got %d: %v", len(entries), entries)
	}
	if _, ok := entries["real"]; !ok {
		t.Fatal("expected key \"real\" in entries")
	}
}

func TestTenantRegistry_ClearByPrefix(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer client.Close()
	reg := NewTenantRegistry[string, string](client, "test:prefix", func(k string) string { return k })
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	_ = reg.Put(ctx, tm, "100:a", "va")
	_ = reg.Put(ctx, tm, "100:b", "vb")
	_ = reg.Put(ctx, tm, "200:a", "vc")

	deleted, err := reg.ClearByPrefix(ctx, tm, "100")
	if err != nil {
		t.Fatalf("ClearByPrefix: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 deleted, got %d", deleted)
	}

	// "100:a" and "100:b" should be gone.
	if ok, _ := reg.Exists(ctx, tm, "100:a"); ok {
		t.Fatal("100:a should be deleted")
	}
	if ok, _ := reg.Exists(ctx, tm, "100:b"); ok {
		t.Fatal("100:b should be deleted")
	}
	// "200:a" should still exist.
	if ok, _ := reg.Exists(ctx, tm, "200:a"); !ok {
		t.Fatal("200:a should still exist")
	}
}

func TestTenantRegistry_ClearByPrefix_Empty(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer client.Close()
	reg := NewTenantRegistry[string, string](client, "test:prefix:empty", func(k string) string { return k })
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	deleted, err := reg.ClearByPrefix(ctx, tm, "999")
	if err != nil {
		t.Fatalf("ClearByPrefix on empty: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("expected 0 deleted, got %d", deleted)
	}
}

// helper for non-shared use; ensures package compiles with strconv import.
var _ = strconv.Itoa
