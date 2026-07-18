package hidden

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return ten
}

func setup(t *testing.T) *Registry {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	r := newRegistry(rc)
	t.Cleanup(func() { r.Clear(context.Background()) })
	return r
}

func TestAddRemoveMemberSet(t *testing.T) {
	r := setup(t)
	ctx := context.Background()
	ten := testTenant(t)

	ms, err := r.MemberSet(ctx, ten)
	if err != nil {
		t.Fatalf("MemberSet: %v", err)
	}
	if len(ms) != 0 {
		t.Fatalf("expected empty set, got %v", ms)
	}

	if err := r.Add(ctx, ten, 42); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Idempotent double-add (FR-1.4).
	if err := r.Add(ctx, ten, 42); err != nil {
		t.Fatalf("Add twice: %v", err)
	}
	ms, _ = r.MemberSet(ctx, ten)
	if _, ok := ms[42]; !ok || len(ms) != 1 {
		t.Fatalf("expected {42}, got %v", ms)
	}

	if err := r.Remove(ctx, ten, 42); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	// Idempotent double-remove (FR-1.4).
	if err := r.Remove(ctx, ten, 42); err != nil {
		t.Fatalf("Remove twice: %v", err)
	}
	ms, _ = r.MemberSet(ctx, ten)
	if len(ms) != 0 {
		t.Fatalf("expected empty after remove, got %v", ms)
	}
}

func TestTenantIsolationAndGetAll(t *testing.T) {
	r := setup(t)
	ctx := context.Background()
	tenA := testTenant(t)
	tenB := testTenant(t)

	_ = r.Add(ctx, tenA, 1)
	_ = r.Add(ctx, tenA, 2)
	_ = r.Add(ctx, tenB, 3)

	msA, _ := r.MemberSet(ctx, tenA)
	if len(msA) != 2 {
		t.Fatalf("tenant A expected 2 members, got %v", msA)
	}
	msB, _ := r.MemberSet(ctx, tenB)
	if _, ok := msB[3]; !ok || len(msB) != 1 {
		t.Fatalf("tenant B expected {3}, got %v", msB)
	}

	all := r.GetAll(ctx)
	if len(all[tenA]) != 2 || len(all[tenB]) != 1 {
		t.Fatalf("GetAll mismatch: %v", all)
	}
}
