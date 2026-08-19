package occurrence

import (
	"atlas-events/event/registry"
	"atlas-events/event/transition"
	"encoding/json"
	"errors"
	"testing"
)

// FR-O6/FR-T2: there is no path that writes one without the other.
func TestCreateWritesOccurrenceAndTransitionTogether(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)

	o, err := p.CreateFromSeed(testDefinition(t, "CRIMSON_BALROG"), registry.Seed{
		Stage:          "ATTACKING",
		Context:        json.RawMessage(`{"routeId":"r"}`),
		WorldId:        1,
		ChannelId:      4,
		ConcurrencyKey: "v1|1|4",
		Maps: []registry.MapScope{
			{MapId: 200090010, Visual: true},
			{MapId: 200090011, Visual: false},
		},
	}, "work-1")
	if err != nil {
		t.Fatalf("CreateFromSeed: %v", err)
	}
	if o.State() != StateActive || o.Stage() != "ATTACKING" {
		t.Fatalf("occurrence = %s/%s", o.State(), o.Stage())
	}

	var trans int64
	db.Model(&transition.Entity{}).Where("occurrence_id = ?", o.Id()).Count(&trans)
	if trans != 1 {
		t.Fatalf("expected 1 transition row, got %d", trans)
	}
	var maps int64
	db.Model(&MapEntity{}).Where("occurrence_id = ?", o.Id()).Count(&maps)
	if maps != 2 {
		t.Fatalf("expected 2 map rows, got %d", maps)
	}
}

// design §5.3 guard 3: even if dedup and the work-row state machine both fail,
// the concurrency key makes the SECOND occurrence insert fail rather than
// producing two live attacks on one voyage.
func TestConcurrencyKeyRejectsASecondActiveOccurrence(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	d := testDefinition(t, "CRIMSON_BALROG")
	seed := registry.Seed{Stage: "ATTACKING", ConcurrencyKey: "v1|1|4", WorldId: 1, ChannelId: 4}

	if _, err := p.CreateFromSeed(d, seed, "work-1"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := p.CreateFromSeed(d, seed, "work-2")
	if !errors.Is(err, ErrConcurrencyKeyTaken) {
		t.Fatalf("second create err = %v, want ErrConcurrencyKeyTaken", err)
	}
}

// task-19 review F1: a COMPLETED occurrence's concurrency key must be
// reusable — the stale ux_event_occurrence_concurrency_key index (no state
// predicate) blocked this forever under Postgres's untargeted
// ON CONFLICT DO NOTHING; SQLite's ON CONFLICT semantics differ, so this
// version of the test cannot itself prove the defect (see
// concurrency_key_integration_test.go, TestConcurrencyKeyIsReusableAfterCompletionOnPostgres,
// for the live Postgres reproduction), but it still pins the correct
// behavior under the default gate.
func TestConcurrencyKeyIsReusableAfterCompletion(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	d := testDefinition(t, "CRIMSON_BALROG")
	seed := registry.Seed{Stage: "ATTACKING", ConcurrencyKey: "v1|1|4", WorldId: 1, ChannelId: 4}

	first, err := p.CreateFromSeed(d, seed, "work-1")
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := p.Complete(first.Id(), "MONSTERS_ELIMINATED", transition.TriggerTypeMonsterKilled, "work-1"); err != nil {
		t.Fatalf("complete: %v", err)
	}

	second, err := p.CreateFromSeed(d, seed, "work-2")
	if err != nil {
		t.Fatalf("re-create after completion: %v, want success", err)
	}
	if second.Id() == first.Id() {
		t.Fatalf("re-create returned the same occurrence id")
	}
}

// task-19 review F1: the old index was also wrong on columns — it omitted
// event_definition_id, so two different definitions sharing a concurrency
// key would collide with each other. ux_occ_concurrency includes
// event_definition_id, so both may hold the same key concurrently.
func TestDifferentDefinitionsMayShareAConcurrencyKey(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	d1 := testDefinition(t, "CRIMSON_BALROG")
	d2 := testDefinition(t, "GOLDEN_BALROG")
	seed := registry.Seed{Stage: "ATTACKING", ConcurrencyKey: "v1|1|4", WorldId: 1, ChannelId: 4}

	if _, err := p.CreateFromSeed(d1, seed, "work-1"); err != nil {
		t.Fatalf("first definition create: %v", err)
	}
	if _, err := p.CreateFromSeed(d2, seed, "work-2"); err != nil {
		t.Fatalf("second definition create: %v, want success", err)
	}
}

// An empty concurrency key opts out of the constraint entirely.
func TestEmptyConcurrencyKeyAllowsMany(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	d := testDefinition(t, "UNBOUNDED")
	seed := registry.Seed{Stage: "", ConcurrencyKey: ""}

	for i := 0; i < 3; i++ {
		if _, err := p.CreateFromSeed(d, seed, "work"); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
}

// FR-B20: two racing completion paths produce exactly one completion and one
// reason. The loser is told it lost, and must NOT run cleanup again.
func TestCompleteIsWonByExactlyOneCaller(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	o, err := p.CreateFromSeed(testDefinition(t, "CRIMSON_BALROG"),
		registry.Seed{Stage: "ATTACKING", ConcurrencyKey: "k"}, "w")
	if err != nil {
		t.Fatalf("CreateFromSeed: %v", err)
	}

	wonA, err := p.Complete(o.Id(), "MONSTERS_ELIMINATED", transition.TriggerTypeMonsterKilled, "u1")
	if err != nil {
		t.Fatalf("Complete A: %v", err)
	}
	wonB, err := p.Complete(o.Id(), "VESSEL_ARRIVED", transition.TriggerTypeVoyageArrived, "v1")
	if err != nil {
		t.Fatalf("Complete B: %v", err)
	}
	if !wonA || wonB {
		t.Fatalf("wonA=%v wonB=%v, want true/false", wonA, wonB)
	}

	final, err := p.GetById(o.Id())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if final.State() != StateCompleted || final.CompletionReason() != "MONSTERS_ELIMINATED" {
		t.Fatalf("final = %s/%s", final.State(), final.CompletionReason())
	}
}

// design §9.5: set semantics, not a counter. A redelivered KILLED must not
// double-decrement, and a KILLED arriving BEFORE its CREATED must not be
// resurrected by the later CREATED.
func TestMonsterTallyIsIdempotentAndOrderIndependent(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	o, _ := p.CreateFromSeed(testDefinition(t, "CRIMSON_BALROG"),
		registry.Seed{Stage: "ATTACKING", ConcurrencyKey: "k"}, "w")

	if err := p.ObserveMonsterSpawned(o.Id(), 1, 8150000); err != nil {
		t.Fatalf("spawned: %v", err)
	}
	// KILLED before CREATED for unique id 2.
	if err := p.ObserveMonsterGone(o.Id(), 2, 8150000); err != nil {
		t.Fatalf("gone: %v", err)
	}
	if err := p.ObserveMonsterSpawned(o.Id(), 2, 8150000); err != nil {
		t.Fatalf("late spawned: %v", err)
	}

	total, alive, err := p.MonsterTally(o.Id())
	if err != nil {
		t.Fatalf("MonsterTally: %v", err)
	}
	if total != 2 || alive != 1 {
		t.Fatalf("total=%d alive=%d, want 2/1 (the late CREATED must not resurrect)", total, alive)
	}

	// Redelivery of both events changes nothing.
	_ = p.ObserveMonsterGone(o.Id(), 2, 8150000)
	_ = p.ObserveMonsterSpawned(o.Id(), 1, 8150000)
	total, alive, _ = p.MonsterTally(o.Id())
	if total != 2 || alive != 1 {
		t.Fatalf("after redelivery total=%d alive=%d, want 2/1", total, alive)
	}
}
