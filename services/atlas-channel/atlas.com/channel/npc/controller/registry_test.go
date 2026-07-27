package controller

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupRegistry(t *testing.T) (*Registry, tenant.Model, field.Model) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(0, 1, 100000000).Build()
	return newRegistry(rc), ten, f
}

func TestClaimIsFirstWriterWins(t *testing.T) {
	r, ten, f := setupRegistry(t)
	ctx := context.Background()

	won, err := r.Claim(ctx, ten, f, 1000, 7)
	if err != nil || !won {
		t.Fatalf("first claim must win: won=%v err=%v", won, err)
	}
	won, err = r.Claim(ctx, ten, f, 1000, 8)
	if err != nil || won {
		t.Fatalf("second claim must lose: won=%v err=%v", won, err)
	}
	cur, ok, err := r.ControllerOf(ctx, ten, f, 1000)
	if err != nil || !ok || cur != 7 {
		t.Fatalf("controller must remain 7: cur=%d ok=%v err=%v", cur, ok, err)
	}
}

func TestReleaseAndAbsence(t *testing.T) {
	r, ten, f := setupRegistry(t)
	ctx := context.Background()

	_, ok, err := r.ControllerOf(ctx, ten, f, 1000)
	if err != nil || ok {
		t.Fatalf("absent entry must report ok=false: ok=%v err=%v", ok, err)
	}

	_, _ = r.Claim(ctx, ten, f, 1000, 7)
	_, _ = r.Claim(ctx, ten, f, 1001, 7)
	_, _ = r.Claim(ctx, ten, f, 1002, 9)

	got, err := r.ControlledBy(ctx, ten, f, 7)
	if err != nil || len(got) != 2 {
		t.Fatalf("expected 2 NPCs controlled by 7, got %v err %v", got, err)
	}

	if err := r.Release(ctx, ten, f, 1000, 1001); err != nil {
		t.Fatalf("Release: %v", err)
	}
	all, _ := r.GetAll(ctx, ten, f)
	if len(all) != 1 || all[1002] != 9 {
		t.Fatalf("expected only 1002->9 left, got %v", all)
	}
	// Idempotent double-release.
	if err := r.Release(ctx, ten, f, 1000); err != nil {
		t.Fatalf("double release must be a no-op: %v", err)
	}
}

func TestTenantAndFieldIsolation(t *testing.T) {
	r, ten, f := setupRegistry(t)
	ctx := context.Background()

	otherTen, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	otherField := field.NewBuilder(0, 1, 200000000).Build()

	won, err := r.Claim(ctx, ten, f, 1000, 7)
	if err != nil || !won {
		t.Fatalf("claim in base tenant/field must win: won=%v err=%v", won, err)
	}

	// Same NPC id in a different tenant must be independent.
	won, err = r.Claim(ctx, otherTen, f, 1000, 42)
	if err != nil || !won {
		t.Fatalf("claim in other tenant must win independently: won=%v err=%v", won, err)
	}

	// Same NPC id in a different field (same tenant) must be independent.
	won, err = r.Claim(ctx, ten, otherField, 1000, 99)
	if err != nil || !won {
		t.Fatalf("claim in other field must win independently: won=%v err=%v", won, err)
	}

	cur, ok, err := r.ControllerOf(ctx, ten, f, 1000)
	if err != nil || !ok || cur != 7 {
		t.Fatalf("base tenant/field controller must remain 7: cur=%d ok=%v err=%v", cur, ok, err)
	}
	cur, ok, err = r.ControllerOf(ctx, otherTen, f, 1000)
	if err != nil || !ok || cur != 42 {
		t.Fatalf("other tenant controller must be 42: cur=%d ok=%v err=%v", cur, ok, err)
	}
	cur, ok, err = r.ControllerOf(ctx, ten, otherField, 1000)
	if err != nil || !ok || cur != 99 {
		t.Fatalf("other field controller must be 99: cur=%d ok=%v err=%v", cur, ok, err)
	}
}
