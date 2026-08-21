package character

import (
	"context"
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// setupMoveTest returns a context carrying a fresh tenant and a minimally
// constructed ProcessorImpl, so cases stay independent (Move keys temporal
// state by characterId within a tenant, but a fresh tenant per test avoids
// any cross-test coupling through the process-singleton registry).
func setupMoveTest(t *testing.T) (context.Context, tenant.Model, *ProcessorImpl) {
	t.Helper()
	setupResourceTestRegistry(t)
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)
	return ctx, tm, &ProcessorImpl{ctx: ctx}
}

func TestMove_ZeroFh_PreservesStoredFoothold(t *testing.T) {
	ctx, tm, p := setupMoveTest(t)

	GetTemporalRegistry().Update(ctx, tm, 42, 10, 20, 77, 3)

	if err := p.Move(42, 300, -50, 0, 5); err != nil {
		t.Fatalf("Move: %v", err)
	}

	got := GetTemporalRegistry().GetById(ctx, tm, 42)
	if got.X() != 300 || got.Y() != -50 || got.Fh() != 77 || got.Stance() != 5 {
		t.Errorf("got x=%d y=%d fh=%d stance=%d, want x=300 y=-50 fh=77 stance=5",
			got.X(), got.Y(), got.Fh(), got.Stance())
	}
}

func TestMove_NonZeroFh_OverwritesFoothold(t *testing.T) {
	ctx, tm, p := setupMoveTest(t)

	GetTemporalRegistry().Update(ctx, tm, 42, 10, 20, 77, 3)

	if err := p.Move(42, 300, -50, 88, 5); err != nil {
		t.Fatalf("Move: %v", err)
	}

	got := GetTemporalRegistry().GetById(ctx, tm, 42)
	if got.X() != 300 || got.Y() != -50 || got.Fh() != 88 || got.Stance() != 5 {
		t.Errorf("got x=%d y=%d fh=%d stance=%d, want x=300 y=-50 fh=88 stance=5",
			got.X(), got.Y(), got.Fh(), got.Stance())
	}
}

func TestMove_ZeroFh_NoPriorState_StoresZeroFh(t *testing.T) {
	ctx, tm, p := setupMoveTest(t)

	if err := p.Move(43, 5, 6, 0, 1); err != nil {
		t.Fatalf("Move: %v", err)
	}

	got := GetTemporalRegistry().GetById(ctx, tm, 43)
	if got.X() != 5 || got.Y() != 6 || got.Fh() != 0 || got.Stance() != 1 {
		t.Errorf("got x=%d y=%d fh=%d stance=%d, want x=5 y=6 fh=0 stance=1",
			got.X(), got.Y(), got.Fh(), got.Stance())
	}
}
