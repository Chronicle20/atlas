package monster

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// forceControlProcessor is recordingProcessor plus the two field/hidden seams
// ForceControl consults. recordingProcessor leaves them nil.
func forceControlProcessor(ctx context.Context, tm tenant.Model, emitted *int, inField []uint32, hidden map[uint32]struct{}) *ProcessorImpl {
	p := recordingProcessor(ctx, tm, emitted)
	p.inFieldFn = func(_ field.Model) ([]uint32, error) { return inField, nil }
	p.hiddenFn = func() (map[uint32]struct{}, error) { return hidden, nil }
	return p
}

func TestForceControl_HandsOverWithAggroFlagSet(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	const (
		previous = uint32(7)
		caster   = uint32(9)
	)
	uid := newAggroedMonster(t, ctx, tm, previous, nil)

	emitted := 0
	p := forceControlProcessor(ctx, tm, &emitted, []uint32{previous, caster}, map[uint32]struct{}{})

	if err := p.ForceControl(uid, caster); err != nil {
		t.Fatalf("ForceControl: %v", err)
	}

	got, err := p.GetById(uid)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.ControlCharacterId() != caster {
		t.Fatalf("ControlCharacterId = %d, want %d", got.ControlCharacterId(), caster)
	}
	if !got.ControllerHasAggro() {
		t.Fatal("ControllerHasAggro = false; the handover must set the flag so START_CONTROL writes StartControlMonsterBody(m, true)")
	}
	// STOP_CONTROL (previous controller) + START_CONTROL (new controller).
	if emitted < 2 {
		t.Fatalf("emitted %d events, want at least 2 (stop then start)", emitted)
	}
}

func TestForceControl_UncontrolledMonsterEmitsStartOnly(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	const caster = uint32(9)
	uid := newAggroedMonster(t, ctx, tm, 0, nil)

	emitted := 0
	p := forceControlProcessor(ctx, tm, &emitted, []uint32{caster}, map[uint32]struct{}{})

	if err := p.ForceControl(uid, caster); err != nil {
		t.Fatalf("ForceControl: %v", err)
	}
	got, err := p.GetById(uid)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.ControlCharacterId() != caster || !got.ControllerHasAggro() {
		t.Fatalf("controller = %d hasAggro = %v, want %d/true", got.ControlCharacterId(), got.ControllerHasAggro(), caster)
	}
}

func TestForceControl_SameControllerIsANoOp(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	const caster = uint32(9)
	uid := newAggroedMonster(t, ctx, tm, caster, nil)

	emitted := 0
	p := forceControlProcessor(ctx, tm, &emitted, []uint32{caster}, map[uint32]struct{}{})

	if err := p.ForceControl(uid, caster); err != nil {
		t.Fatalf("ForceControl: %v", err)
	}
	if emitted != 0 {
		t.Fatalf("emitted %d events, want 0 — forcing control to the current controller must not emit a redundant control packet (FR-5.4)", emitted)
	}
}

func TestForceControl_CharacterNotInFieldIsDropped(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	uid := newAggroedMonster(t, ctx, tm, 7, nil)

	emitted := 0
	p := forceControlProcessor(ctx, tm, &emitted, []uint32{7}, map[uint32]struct{}{})

	if err := p.ForceControl(uid, 9); err != nil {
		t.Fatalf("ForceControl for an absent character must be dropped, not an error: %v", err)
	}
	if emitted != 0 {
		t.Fatalf("emitted %d events, want 0", emitted)
	}
}

func TestForceControl_HiddenCharacterIsDropped(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	uid := newAggroedMonster(t, ctx, tm, 7, nil)

	emitted := 0
	p := forceControlProcessor(ctx, tm, &emitted, []uint32{7, 9}, map[uint32]struct{}{9: {}})

	if err := p.ForceControl(uid, 9); err != nil {
		t.Fatalf("ForceControl for a GM-hidden character must be dropped: %v", err)
	}
	if emitted != 0 {
		t.Fatalf("emitted %d events, want 0 — RelinquishControlOnHide would immediately strip it back, producing a flap", emitted)
	}
}

func TestForceControl_MissingMonsterIsDropped(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	emitted := 0
	p := forceControlProcessor(ctx, tm, &emitted, []uint32{9}, map[uint32]struct{}{})

	if err := p.ForceControl(4242, 9); err != nil {
		t.Fatalf("ForceControl on a nonexistent monster must be dropped: %v", err)
	}
	if emitted != 0 {
		t.Fatalf("emitted %d events, want 0", emitted)
	}
}

// TestStartControlStillDefaultsAggroOff guards the forceAggro split: the
// existing StartControl path must be byte-for-byte unchanged in behaviour.
func TestStartControlStillDefaultsAggroOff(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	uid := newAggroedMonster(t, ctx, tm, 0, nil)

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)

	m, err := p.StartControl(uid, 9)
	if err != nil {
		t.Fatalf("StartControl: %v", err)
	}
	if m.ControllerHasAggro() {
		t.Fatal("StartControl set the aggro flag; only ForceControl may do that")
	}
	if m.ControlCharacterId() != 9 {
		t.Fatalf("ControlCharacterId = %d, want 9", m.ControlCharacterId())
	}
}
