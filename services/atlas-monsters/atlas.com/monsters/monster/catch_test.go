package monster

import (
	"atlas-monsters/monster/consumable"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// withCatchItem installs the test-only consumable lookup and returns a cleanup.
func withCatchItem(t *testing.T, m consumable.Model, err error) func() {
	t.Helper()
	prev := testConsumableLookup
	testConsumableLookup = func(_ uint32) (consumable.Model, error) { return m, err }
	return func() { testConsumableLookup = prev }
}

func spawnCatchable(t *testing.T, ten tenant.Model, monsterId uint32, hp uint32, maxHp uint32) uint32 {
	t.Helper()
	r := GetMonsterRegistry()
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(context.Background(), ten, f, monsterId, 0, 0, 0, 5, 0, maxHp, 100)
	if hp != maxHp {
		if _, err := r.ApplyDamage(ten, 1, maxHp-hp, m.UniqueId(), time.Now().UnixMilli()); err != nil {
			t.Fatalf("apply damage: %v", err)
		}
	}
	return m.UniqueId()
}

func eventTypes(events *[]emittedBody) []string {
	var out []string
	for _, e := range *events {
		out = append(out, e.Type)
	}
	return out
}

// TestCatch_Success — species matches, HP under the gate, no bridleProp (so the
// roll is a deterministic pass): the monster is claimed and removed, and the
// three events fire in the order CATCH_RESOLVED, CAUGHT, DESTROYED.
func TestCatch_Success(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	GetMonsterRegistry().Clear(context.Background())
	defer withCatchItem(t, consumable.NewModelBuilder().
		SetId(2270000).SetMonsterId(9300101).SetCreate(1902000).Build(), nil)()

	uniqueId := spawnCatchable(t, ten, 9300101, 100, 100)
	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Catch(uniqueId, 42, 2270000)

	if got := eventTypes(events); len(got) != 3 ||
		got[0] != EventMonsterCatchResolved || got[1] != EventMonsterStatusCaught || got[2] != EventMonsterStatusDestroyed {
		t.Fatalf("event order = %v, want [CATCH_RESOLVED CAUGHT DESTROYED]", eventTypes(events))
	}
	if (*events)[0].Topic != EnvEventTopicMonsterCatch {
		t.Errorf("CATCH_RESOLVED topic = %q, want %q", (*events)[0].Topic, EnvEventTopicMonsterCatch)
	}
	var body catchResolvedBody
	if err := json.Unmarshal((*events)[0].Body, &body); err != nil {
		t.Fatalf("decode CATCH_RESOLVED: %v", err)
	}
	if !body.Success || body.CharacterId != 42 || body.ItemId != 2270000 || body.Cause != "" {
		t.Fatalf("CATCH_RESOLVED body = %+v", body)
	}
	if _, err := GetMonsterRegistry().GetMonster(ten, uniqueId); err == nil {
		t.Error("monster still present after a successful catch")
	}
}

// TestCatch_SpeciesMismatch — the item names a different mob: failure, monster
// untouched, no KILLED and no DESTROYED (no experience, no drops, no death).
func TestCatch_SpeciesMismatch(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	GetMonsterRegistry().Clear(context.Background())
	defer withCatchItem(t, consumable.NewModelBuilder().
		SetId(2270000).SetMonsterId(9300101).SetCreate(1902000).Build(), nil)()

	uniqueId := spawnCatchable(t, ten, 9500197, 100, 100)
	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Catch(uniqueId, 42, 2270000)

	if got := eventTypes(events); len(got) != 2 ||
		got[0] != EventMonsterCatchResolved || got[1] != EventMonsterStatusCatchFailed {
		t.Fatalf("event order = %v, want [CATCH_RESOLVED CATCH_FAILED]", eventTypes(events))
	}
	var body catchResolvedBody
	_ = json.Unmarshal((*events)[0].Body, &body)
	if body.Success || body.Cause != CatchCauseSpeciesMismatch {
		t.Fatalf("CATCH_RESOLVED body = %+v, want success=false cause=SPECIES_MISMATCH", body)
	}
	if _, err := GetMonsterRegistry().GetMonster(ten, uniqueId); err != nil {
		t.Error("monster removed on a species mismatch")
	}
}

// TestCatch_HpGate — mobHP is a PERCENTAGE of max HP (design A-1). The
// cross-multiplied comparison must admit hp exactly at the boundary and reject
// one point above it, and mobHP=100 must admit a FULL-HP monster (the case
// integer truncation would break).
func TestCatch_HpGate(t *testing.T) {
	cases := []struct {
		name   string
		mobHP  uint32
		hp     uint32
		maxHp  uint32
		expect bool
	}{
		{"at the 40% boundary", 40, 400, 1000, true},
		{"one point above 40%", 40, 401, 1000, false},
		{"well under 30%", 30, 100, 1000, true},
		{"full HP at mobHP=100", 100, 1000, 1000, true},
		{"no gate when mobHP is zero", 0, 1000, 1000, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			GetMonsterRegistry().Clear(context.Background())
			defer withCatchItem(t, consumable.NewModelBuilder().
				SetId(2270005).SetMonsterId(9300187).SetCreate(2109001).SetMonsterHp(tc.mobHP).Build(), nil)()

			uniqueId := spawnCatchable(t, ten, 9300187, tc.hp, tc.maxHp)
			p, events := newRecordingProcessorWithBodies(t, ten)
			p.Catch(uniqueId, 42, 2270005)

			var body catchResolvedBody
			_ = json.Unmarshal((*events)[0].Body, &body)
			if body.Success != tc.expect {
				t.Fatalf("success = %t, want %t (cause %q)", body.Success, tc.expect, body.Cause)
			}
			if !tc.expect && body.Cause != CatchCauseHpTooHigh {
				t.Fatalf("cause = %q, want HP_TOO_HIGH", body.Cause)
			}
		})
	}
}

// TestCatch_RollFailed — a seeded roller that always loses produces ROLL_FAILED
// and leaves the monster alive.
func TestCatch_RollFailed(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	GetMonsterRegistry().Clear(context.Background())
	defer withCatchItem(t, consumable.NewModelBuilder().
		SetId(2270002).SetMonsterId(9300157).SetCreate(4031868).SetBridleProp(50).SetBridlePropChg(1.2).Build(), nil)()

	prevRoll := testCatchRoll
	testCatchRoll = func(chance uint32) (bool, error) {
		if chance != 60 {
			t.Errorf("effective chance = %d, want 60 (50 * 1.2, rounded)", chance)
		}
		return false, nil
	}
	defer func() { testCatchRoll = prevRoll }()

	uniqueId := spawnCatchable(t, ten, 9300157, 100, 100)
	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Catch(uniqueId, 42, 2270002)

	var body catchResolvedBody
	_ = json.Unmarshal((*events)[0].Body, &body)
	if body.Success || body.Cause != CatchCauseRollFailed {
		t.Fatalf("CATCH_RESOLVED body = %+v, want ROLL_FAILED", body)
	}
	if _, err := GetMonsterRegistry().GetMonster(ten, uniqueId); err != nil {
		t.Error("monster removed on a failed roll")
	}
}

// TestCatch_MonsterGone — a redelivered command whose monster is already gone
// emits CATCH_RESOLVED(false, UNRESOLVED) so the reservation is cancelled, and
// CATCH_FAILED(UNRESOLVED) so the client unlocks. It grants nothing.
func TestCatch_MonsterGone(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	GetMonsterRegistry().Clear(context.Background())
	defer withCatchItem(t, consumable.NewModelBuilder().
		SetId(2270000).SetMonsterId(9300101).SetCreate(1902000).Build(), nil)()

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Catch(999999, 42, 2270000)

	if got := eventTypes(events); len(got) != 2 ||
		got[0] != EventMonsterCatchResolved || got[1] != EventMonsterStatusCatchFailed {
		t.Fatalf("event order = %v, want [CATCH_RESOLVED CATCH_FAILED]", eventTypes(events))
	}
	var body catchResolvedBody
	_ = json.Unmarshal((*events)[0].Body, &body)
	if body.Success || body.Cause != CatchCauseUnresolved {
		t.Fatalf("CATCH_RESOLVED body = %+v, want UNRESOLVED", body)
	}
}

// TestCatch_ConcurrentAttempts_OneCaught — two players catching the same monster
// concurrently must produce exactly one CAUGHT (NFR-Race-safety). The loser
// reports UNRESOLVED, which cancels its reservation and unlocks its client
// without a failure notice. Each processor records into its own event slice, so
// the recorder itself is not shared state.
func TestCatch_ConcurrentAttempts_OneCaught(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	GetMonsterRegistry().Clear(context.Background())
	defer withCatchItem(t, consumable.NewModelBuilder().
		SetId(2270000).SetMonsterId(9300101).SetCreate(1902000).Build(), nil)()

	uniqueId := spawnCatchable(t, ten, 9300101, 100, 100)

	const racers = 8
	recorders := make([]*[]emittedBody, racers)
	var wg sync.WaitGroup
	wg.Add(racers)
	for i := 0; i < racers; i++ {
		p, events := newRecordingProcessorWithBodies(t, ten)
		recorders[i] = events
		go func(p *ProcessorImpl, characterId uint32) {
			defer wg.Done()
			p.Catch(uniqueId, characterId, 2270000)
		}(p, uint32(100+i))
	}
	wg.Wait()

	caught := 0
	for _, events := range recorders {
		for _, e := range *events {
			if e.Type == EventMonsterStatusCaught {
				caught++
			}
		}
	}
	if caught != 1 {
		t.Fatalf("CAUGHT events = %d, want exactly 1", caught)
	}
}

// TestCatch_LookupFailure — fail-closed, exactly like Kill's boss lookup: a data
// error drops the command entirely rather than reporting a catch failure.
func TestCatch_LookupFailure(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	GetMonsterRegistry().Clear(context.Background())
	defer withCatchItem(t, consumable.Model{}, errors.New("upstream down"))()

	uniqueId := spawnCatchable(t, ten, 9300101, 100, 100)
	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Catch(uniqueId, 42, 2270000)

	if len(*events) != 0 {
		t.Fatalf("expected no events on a data lookup failure, got %v", eventTypes(events))
	}
	if _, err := GetMonsterRegistry().GetMonster(ten, uniqueId); err != nil {
		t.Error("monster removed despite a data lookup failure")
	}
}
