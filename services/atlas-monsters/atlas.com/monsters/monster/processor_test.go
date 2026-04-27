package monster

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// emittedEvent captures a single Kafka message emitted during a test.
type emittedEvent struct {
	Topic string
	Type  string
}

// newRecordingProcessor constructs a ProcessorImpl with a stub emitter that
// records every emitted event so tests can assert on ordering and topic.
func newRecordingProcessor(t *testing.T, ten tenant.Model) (*ProcessorImpl, *[]emittedEvent) {
	t.Helper()
	var events []emittedEvent
	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: context.Background(),
		t:   ten,
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			msgs, err := provider()
			if err != nil {
				t.Fatalf("provider error: %v", err)
			}
			for _, m := range msgs {
				var env struct {
					Type string `json:"type"`
				}
				if err := json.Unmarshal(m.Value, &env); err != nil {
					t.Fatalf("decode emitted message: %v", err)
				}
				events = append(events, emittedEvent{Topic: topic, Type: env.Type})
			}
			return nil
		},
	}
	p.inFieldFn = func(_ field.Model) ([]uint32, error) {
		return nil, nil
	}
	return p, &events
}

// TestDamageMultiLineKillOnLastLine verifies that when the killing blow is the
// final damage line (40+30+50=120 vs 100 HP), both a DAMAGED event and a
// KILLED event are emitted in that order and the monster is removed from the
// registry.
func TestDamageMultiLineKillOnLastLine(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	monsterId := uint32(9300018)
	m := r.CreateMonster(ctx, ten, f, monsterId, 0, 0, 0, 5, 0, 100, 50)
	uniqueId := m.UniqueId()

	charId := uint32(1)
	p, events := newRecordingProcessor(t, ten)
	p.Damage(uniqueId, charId, []uint32{40, 30, 50}, 0)

	if len(*events) != 2 {
		t.Fatalf("expected 2 events (damaged+killed), got %d: %v", len(*events), *events)
	}
	if (*events)[0].Type != EventMonsterStatusDamaged {
		t.Errorf("event[0]: expected %q, got %q", EventMonsterStatusDamaged, (*events)[0].Type)
	}
	if (*events)[1].Type != EventMonsterStatusKilled {
		t.Errorf("event[1]: expected %q, got %q", EventMonsterStatusKilled, (*events)[1].Type)
	}
	if (*events)[0].Topic != EnvEventTopicMonsterStatus || (*events)[1].Topic != EnvEventTopicMonsterStatus {
		t.Errorf("events emitted to wrong topic: %v", *events)
	}

	// Monster must have been removed from the registry.
	if _, err := r.GetMonster(ten, uniqueId); err == nil {
		t.Error("expected monster to be removed from registry after kill, but GetMonster succeeded")
	}
}

// TestDamageMultiLineKillOnMiddleLine verifies that when the killing blow is
// the second of three lines (40+80=120 vs 100 HP), the third line is NOT
// applied and only 2 events are emitted.
func TestDamageMultiLineKillOnMiddleLine(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	monsterId := uint32(9300018)
	m := r.CreateMonster(ctx, ten, f, monsterId, 0, 0, 0, 5, 0, 100, 50)
	uniqueId := m.UniqueId()

	charId := uint32(1)
	p, events := newRecordingProcessor(t, ten)
	// Line 1: 40 damage (HP→60), Line 2: 80 damage (kills; HP→0), Line 3: 50 (must NOT apply)
	p.Damage(uniqueId, charId, []uint32{40, 80, 50}, 0)

	if len(*events) != 2 {
		t.Fatalf("expected 2 events (damaged+killed), got %d: %v", len(*events), *events)
	}
	if (*events)[0].Type != EventMonsterStatusDamaged {
		t.Errorf("event[0]: expected %q, got %q", EventMonsterStatusDamaged, (*events)[0].Type)
	}
	if (*events)[1].Type != EventMonsterStatusKilled {
		t.Errorf("event[1]: expected %q, got %q", EventMonsterStatusKilled, (*events)[1].Type)
	}

	// Monster must be gone.
	if _, err := r.GetMonster(ten, uniqueId); err == nil {
		t.Error("expected monster to be removed from registry after kill, but GetMonster succeeded")
	}
}

// TestDamageSingleLineKill verifies that a one-line killing attack emits both
// DAMAGED and KILLED (in that order). This is a deliberate behavior change
// from pre-task-030 code where a one-line kill emitted only KILLED.
func TestDamageSingleLineKill(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	monsterId := uint32(9300018)
	m := r.CreateMonster(ctx, ten, f, monsterId, 0, 0, 0, 5, 0, 50, 50)
	uniqueId := m.UniqueId()

	charId := uint32(1)
	p, events := newRecordingProcessor(t, ten)
	p.Damage(uniqueId, charId, []uint32{200}, 0)

	if len(*events) != 2 {
		t.Fatalf("expected 2 events (damaged+killed), got %d: %v", len(*events), *events)
	}
	if (*events)[0].Type != EventMonsterStatusDamaged {
		t.Errorf("event[0]: expected %q, got %q", EventMonsterStatusDamaged, (*events)[0].Type)
	}
	if (*events)[1].Type != EventMonsterStatusKilled {
		t.Errorf("event[1]: expected %q, got %q", EventMonsterStatusKilled, (*events)[1].Type)
	}

	// Monster must be gone.
	if _, err := r.GetMonster(ten, uniqueId); err == nil {
		t.Error("expected monster to be removed from registry after kill, but GetMonster succeeded")
	}
}

// TestDamageEmptySlice verifies that an empty damages slice results in zero
// events and the monster is left untouched.
func TestDamageEmptySlice(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, uint32(9300018), 0, 0, 0, 5, 0, 100, 50)
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessor(t, ten)
	p.Damage(uniqueId, 1, []uint32{}, 0)

	if len(*events) != 0 {
		t.Fatalf("expected 0 events for empty damages, got %d: %v", len(*events), *events)
	}
	// Monster must still exist and be at full HP.
	got, err := r.GetMonster(ten, uniqueId)
	if err != nil {
		t.Fatalf("expected monster to remain in registry, got error: %v", err)
	}
	if got.Hp() != 100 {
		t.Errorf("expected monster HP=100 (unchanged), got %d", got.Hp())
	}
}

// TestDamageAlreadyDeadMonster verifies that Damage against a monster with
// HP=0 emits no events.
func TestDamageAlreadyDeadMonster(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	// Create with HP=1, then kill it directly via the registry so HP=0.
	m := r.CreateMonster(ctx, ten, f, uint32(9300018), 0, 0, 0, 5, 0, 1, 50)
	uniqueId := m.UniqueId()
	// Apply a killing hit directly via the registry (no emit).
	r.ApplyDamage(ten, 1, 999, uniqueId, time.Now().UnixMilli())
	// Do NOT remove the monster — just leave it at HP=0 in the registry so
	// Processor.Damage hits the !m.Alive() early-return path.

	p, events := newRecordingProcessor(t, ten)
	p.Damage(uniqueId, 1, []uint32{100}, 0)

	if len(*events) != 0 {
		t.Fatalf("expected 0 events for already-dead monster, got %d: %v", len(*events), *events)
	}
}

// helpers used by the new tests
type emittedBody struct {
	Topic string
	Type  string
	Body  json.RawMessage
}

func newRecordingProcessorWithBodies(t *testing.T, ten tenant.Model) (*ProcessorImpl, *[]emittedBody) {
	t.Helper()
	var events []emittedBody
	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: context.Background(),
		t:   ten,
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			msgs, err := provider()
			if err != nil {
				t.Fatalf("provider error: %v", err)
			}
			for _, m := range msgs {
				var env struct {
					Type string          `json:"type"`
					Body json.RawMessage `json:"body"`
				}
				if err := json.Unmarshal(m.Value, &env); err != nil {
					t.Fatalf("decode emitted: %v", err)
				}
				events = append(events, emittedBody{Topic: topic, Type: env.Type, Body: env.Body})
			}
			return nil
		},
		inFieldFn: func(_ field.Model) ([]uint32, error) {
			return []uint32{1, 2, 3, 4, 7}, nil
		},
	}
	return p, &events
}

// TestDamageControllerSwitchOnDpsLead — character 2 takes lead from character 1
// (the current controller). Expect STOP_CONTROL (for 1) then START_CONTROL
// (for 2) with controllerHasAggro=true. AGGRO_CHANGED suppressed because the
// switch carries the flag.
func TestDamageControllerSwitchOnDpsLead(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	uniqueId := m.UniqueId()
	// Pre-populate: character 1 controls and leads damage.
	if _, err := r.ControlMonster(ten, uniqueId, 1); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	if _, err := r.ApplyDamage(ten, 1, 50, uniqueId, time.Now().UnixMilli()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Damage(uniqueId, 2, []uint32{500}, 0)

	var types []string
	for _, e := range *events {
		types = append(types, e.Type)
	}
	// Expected order: DAMAGED, NEXT_SKILL_DECIDED (damage trigger), STOP_CONTROL,
	// START_CONTROL, NEXT_SKILL_DECIDED (controller-change trigger).
	if len(types) != 5 ||
		types[0] != EventMonsterStatusDamaged ||
		types[1] != EventMonsterStatusNextSkillDecided ||
		types[2] != EventMonsterStatusStopControl ||
		types[3] != EventMonsterStatusStartControl ||
		types[4] != EventMonsterStatusNextSkillDecided {
		t.Fatalf("unexpected event order: %v", types)
	}
	// START_CONTROL body must carry controllerHasAggro=true.
	var body statusEventStartControlBody
	if err := json.Unmarshal((*events)[3].Body, &body); err != nil {
		t.Fatalf("decode start control: %v", err)
	}
	if !body.ControllerHasAggro {
		t.Errorf("START_CONTROL body controllerHasAggro=true expected, got false")
	}
	if body.ActorId != 2 {
		t.Errorf("START_CONTROL ActorId=2 expected, got %d", body.ActorId)
	}
}

// TestDamageNoSwitchWhenLeaderUnchanged — current controller takes more damage
// and stays leader. No STOP/START, but AGGRO_CHANGED should fire (first hit
// flips the flag).
func TestDamageNoSwitchWhenLeaderUnchanged(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	uniqueId := m.UniqueId()
	// Controller is set; controllerHasAggro starts false.
	if _, err := r.ControlMonster(ten, uniqueId, 1); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Damage(uniqueId, 1, []uint32{30}, 0)

	var types []string
	for _, e := range *events {
		types = append(types, e.Type)
	}
	// Expected order: DAMAGED, NEXT_SKILL_DECIDED (damage trigger), AGGRO_CHANGED.
	if len(types) != 3 ||
		types[0] != EventMonsterStatusDamaged ||
		types[1] != EventMonsterStatusNextSkillDecided ||
		types[2] != EventMonsterStatusAggroChanged {
		t.Fatalf("expected DAMAGED + NEXT_SKILL_DECIDED + AGGRO_CHANGED, got %v", types)
	}
	var body statusEventAggroChangedBody
	if err := json.Unmarshal((*events)[2].Body, &body); err != nil {
		t.Fatalf("decode aggro changed: %v", err)
	}
	if body.ControllerCharacterId != 1 || !body.ControllerHasAggro {
		t.Errorf("AGGRO_CHANGED body unexpected: %+v", body)
	}
}

// TestDamageAggroChangedSuppressedOnSwitch — when first hit also triggers a
// controller switch, AGGRO_CHANGED is NOT emitted.
func TestDamageAggroChangedSuppressedOnSwitch(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	uniqueId := m.UniqueId()
	if _, err := r.ControlMonster(ten, uniqueId, 1); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}

	p, events := newRecordingProcessorWithBodies(t, ten)
	// Character 2 hits first AND becomes leader (char 1 has no seed damage, so
	// char 2's 500 damage immediately takes the DPS lead → controller switch).
	p.Damage(uniqueId, 2, []uint32{500}, 0)

	var types []string
	for _, e := range *events {
		types = append(types, e.Type)
	}
	// Expected order: DAMAGED, NEXT_SKILL_DECIDED (damage trigger), STOP_CONTROL,
	// START_CONTROL, NEXT_SKILL_DECIDED (controller-change trigger).
	// AGGRO_CHANGED must NOT appear because the switch carries the aggro flag.
	if len(types) != 5 ||
		types[0] != EventMonsterStatusDamaged ||
		types[1] != EventMonsterStatusNextSkillDecided ||
		types[2] != EventMonsterStatusStopControl ||
		types[3] != EventMonsterStatusStartControl ||
		types[4] != EventMonsterStatusNextSkillDecided {
		t.Fatalf("unexpected event sequence (AGGRO_CHANGED must be suppressed on switch): %v", types)
	}
}

// TestDamageFR9NoStopWhenControllerZero — controller is 0; first attacker
// becomes controller via a single START_CONTROL with no preceding STOP_CONTROL.
func TestDamageFR9NoStopWhenControllerZero(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Damage(uniqueId, 7, []uint32{30}, 0)

	for _, e := range *events {
		if e.Type == EventMonsterStatusStopControl {
			t.Fatalf("STOP_CONTROL must NOT precede START_CONTROL when controller was 0")
		}
	}
	// We expect DAMAGED + START_CONTROL (first hit on monster with no controller
	// keeps WasFirstHit=false, so no AGGRO_CHANGED at this stage).
	var saw bool
	for _, e := range *events {
		if e.Type == EventMonsterStatusStartControl {
			saw = true
		}
	}
	if !saw {
		t.Errorf("expected START_CONTROL, got %v", *events)
	}
}

// TestDamageFR10OutOfFieldSkipsSwitch — attacker not in field: damage applies,
// controller is NOT switched.
func TestDamageFR10OutOfFieldSkipsSwitch(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	uniqueId := m.UniqueId()
	if _, err := r.ControlMonster(ten, uniqueId, 1); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	// Seed the existing controller as leader.
	if _, err := r.ApplyDamage(ten, 1, 50, uniqueId, time.Now().UnixMilli()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	p, events := newRecordingProcessorWithBodies(t, ten)
	// Override inFieldFn so character 2 is NOT in field.
	p.inFieldFn = func(_ field.Model) ([]uint32, error) {
		return []uint32{1}, nil
	}
	p.Damage(uniqueId, 2, []uint32{500}, 0)

	for _, e := range *events {
		if e.Type == EventMonsterStatusStopControl || e.Type == EventMonsterStatusStartControl {
			t.Fatalf("FR-10: out-of-field attacker should not switch controller, got %s", e.Type)
		}
	}
	// Damage still applied (1000 initial - 50 seed - 500 = 450).
	got, _ := r.GetMonster(ten, uniqueId)
	if got.Hp() != 450 {
		t.Errorf("expected HP=450 after seed+500 damage, got %d", got.Hp())
	}
}

// TestAttackerInField verifies the FR-10 helper:
//   - returns true when the attacker's id is in the field's character id list
//   - returns false when not
//   - returns false (fail-closed) on provider error
func TestAttackerInField(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	tests := []struct {
		name   string
		ids    []uint32
		err    error
		wantIn bool
	}{
		{"in field", []uint32{1, 7, 9}, nil, true},
		{"not in field", []uint32{1, 9}, nil, false},
		{"empty field", []uint32{}, nil, false},
		{"provider error fails closed", nil, errors.New("boom"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &ProcessorImpl{
				l:   logrus.New(),
				ctx: ctx,
				t:   ten,
				inFieldFn: func(_ field.Model) ([]uint32, error) {
					return tc.ids, tc.err
				},
			}
			got, err := p.attackerInField(f, 7)
			if tc.err != nil {
				if err == nil {
					t.Errorf("expected error from helper, got nil")
				}
			}
			if got != tc.wantIn {
				t.Errorf("attackerInField=%v want %v", got, tc.wantIn)
			}
		})
	}
}

// TestApplyAnimationDelayedEffect_DeadMonsterSkipsExecute verifies that
// applyAnimationDelayedEffect skips both the executeEffect and postExecute
// closures when the monster is dead (HP=0) at time of invocation.
func TestApplyAnimationDelayedEffect_DeadMonsterSkipsExecute(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, tm, f, 9000000, 0, 0, 0, 0, 0, 100, 50)
	// Mark the monster dead by zeroing its HP directly in the registry.
	dead := Clone(m).SetHp(0).Build()
	r.UpdateMonster(tm, m.UniqueId(), dead)

	executed, posted := false, false
	p := &ProcessorImpl{
		l:    logrus.New(),
		ctx:  ctx,
		t:    tm,
		emit: func(string, model.Provider[[]kafka.Message]) error { return nil },
	}
	p.applyAnimationDelayedEffect(m.UniqueId(), func() { executed = true }, func() { posted = true })

	if executed || posted {
		t.Fatalf("dead monster should skip both execute (%v) and postExecute (%v)", executed, posted)
	}
}

// TestApplyAnimationDelayedEffect_AliveMonsterRunsBoth verifies that
// applyAnimationDelayedEffect runs both the executeEffect and postExecute
// closures when the monster is alive.
func TestApplyAnimationDelayedEffect_AliveMonsterRunsBoth(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, tm, f, 9000000, 0, 0, 0, 0, 0, 100, 50)

	executed, posted := false, false
	p := &ProcessorImpl{
		l:    logrus.New(),
		ctx:  ctx,
		t:    tm,
		emit: func(string, model.Provider[[]kafka.Message]) error { return nil },
	}
	p.applyAnimationDelayedEffect(m.UniqueId(), func() { executed = true }, func() { posted = true })

	if !executed || !posted {
		t.Fatalf("alive monster should run both execute (%v) and postExecute (%v)", executed, posted)
	}
}

// TestDamage_TriggersRepick verifies that a non-killing damage call that changes
// the monster's HP percentage emits at least one NEXT_SKILL_DECIDED event on
// the monster-status topic, confirming the damage repick trigger fires.
func TestDamage_TriggersRepick(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, uint32(9300018), 0, 0, 0, 5, 0, 1000, 50)
	uniqueId := m.UniqueId()

	countByType := make(map[string]int)
	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: context.Background(),
		t:   ten,
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			msgs, err := provider()
			if err != nil {
				t.Fatalf("provider error: %v", err)
			}
			for _, msg := range msgs {
				var env struct {
					Type string `json:"type"`
				}
				if err := json.Unmarshal(msg.Value, &env); err != nil {
					t.Fatalf("decode emitted message: %v", err)
				}
				countByType[env.Type]++
			}
			return nil
		},
		inFieldFn: func(_ field.Model) ([]uint32, error) {
			return nil, nil
		},
	}

	p.Damage(uniqueId, 1, []uint32{999}, 0)

	if countByType[EventMonsterStatusNextSkillDecided] != 1 {
		t.Errorf("expected exactly 1 NEXT_SKILL_DECIDED event from damage trigger; got %d (all events: %v)",
			countByType[EventMonsterStatusNextSkillDecided], countByType)
	}
}
