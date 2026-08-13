package monster

import (
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// newAggroedMonster stands up a monster in the registry, applies damage from
// each of the given characters (which flips controllerHasAggro true once the
// monster is controlled), and returns its unique id.
func newAggroedMonster(t *testing.T, ctx context.Context, tm tenant.Model, controllerId uint32, attackers []uint32) uint32 {
	t.Helper()
	r := GetMonsterRegistry()
	r.CreateMonster(ctx, tm, testField(), 9000000, 0, 0, 0, 0, 0, 100000, 50)
	mons := r.GetMonstersInMap(tm, testField())
	if len(mons) != 1 {
		t.Fatalf("expected 1 monster; got %d", len(mons))
	}
	uid := mons[0].UniqueId()

	if controllerId != 0 {
		if _, err := r.ControlMonster(tm, uid, controllerId); err != nil {
			t.Fatalf("ControlMonster: %v", err)
		}
	}
	for i, a := range attackers {
		if _, err := r.ApplyDamage(tm, a, uint32(100*(i+1)), uid, int64(1_000+i)); err != nil {
			t.Fatalf("ApplyDamage(%d): %v", a, err)
		}
	}
	return uid
}

func TestClearAggro_WipesEveryEntryAndFlipsFlag(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	const controller = uint32(7)
	uid := newAggroedMonster(t, ctx, tm, controller, []uint32{7, 8, 9})

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)

	before, err := p.GetById(uid)
	if err != nil {
		t.Fatalf("GetById before: %v", err)
	}
	if len(before.DamageEntries()) != 3 {
		t.Fatalf("setup: expected 3 damage entries, got %d", len(before.DamageEntries()))
	}
	if !before.ControllerHasAggro() {
		t.Fatal("setup: expected controllerHasAggro true after damage from the controller")
	}

	if err := p.ClearAggro(uid); err != nil {
		t.Fatalf("ClearAggro: %v", err)
	}

	after, err := p.GetById(uid)
	if err != nil {
		t.Fatalf("GetById after: %v", err)
	}
	if len(after.DamageEntries()) != 0 {
		t.Fatalf("DamageEntries = %d, want 0 — the wipe must remove EVERY character's entry, not just the caster's", len(after.DamageEntries()))
	}
	if after.ControllerHasAggro() {
		t.Fatal("ControllerHasAggro = true, want false after a wipe")
	}
	// Losing aggro is not losing control — DecayDamageEntries behaves the same.
	if after.ControlCharacterId() != controller {
		t.Fatalf("ControlCharacterId = %d, want %d; the wipe must not clear the controller",
			after.ControlCharacterId(), controller)
	}
	if emitted != 1 {
		t.Fatalf("emitted %d MONSTER_STATUS events, want exactly 1 (the AGGRO_CHANGED flip)", emitted)
	}
}

func TestClearAggro_EmptyTableIsANoOp(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	uid := newAggroedMonster(t, ctx, tm, 0, nil)

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)

	if err := p.ClearAggro(uid); err != nil {
		t.Fatalf("first ClearAggro: %v", err)
	}
	if err := p.ClearAggro(uid); err != nil {
		t.Fatalf("second ClearAggro must be idempotent: %v", err)
	}
	if emitted != 0 {
		t.Fatalf("emitted %d events, want 0 — wiping an already-empty table must emit nothing", emitted)
	}
}

func TestClearAggro_MissingMonsterIsDropped(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)

	if err := p.ClearAggro(4242); err != nil {
		t.Fatalf("ClearAggro on a nonexistent monster must be dropped, not an error: %v", err)
	}
	if emitted != 0 {
		t.Fatalf("emitted %d events, want 0", emitted)
	}
}

// TestClearAggro_DecaySweepSeesNothingToDo is the FR-4.4 interaction check: the
// wipe converges on the same state DecayDamageEntries reaches when its list
// empties, so the next sweep tick over the same monster is a no-op.
func TestClearAggro_DecaySweepSeesNothingToDo(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	uid := newAggroedMonster(t, ctx, tm, 7, []uint32{7, 8})

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)
	if err := p.ClearAggro(uid); err != nil {
		t.Fatalf("ClearAggro: %v", err)
	}

	summary, err := GetMonsterRegistry().DecayDamageEntries(tm, uid, 1_000_000)
	if err != nil {
		t.Fatalf("DecayDamageEntries after wipe: %v", err)
	}
	if summary.AggroFlippedOff {
		t.Fatal("the decay sweep flipped the aggro flag again after a wipe; the wipe must already have converged")
	}
	if len(summary.Monster.DamageEntries()) != 0 {
		t.Fatalf("decay found %d entries after a wipe, want 0", len(summary.Monster.DamageEntries()))
	}
}
