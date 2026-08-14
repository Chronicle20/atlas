package tasks

import (
	"atlas-maps/mist"
	"atlas-maps/monster"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	mistKafka "atlas-maps/kafka/message/mist"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// TestMistTick_MonsterTarget_SlowLookupDoesNotBlockSiblingMist proves the
// per-tenant fan-out (processTenant / mistTenantConcurrency) actually
// isolates latency, not just correctness.
//
// This is a two-way handshake, not a one-sided block-and-poll: the "blocked"
// mist signals `entered` the instant its lookup starts and only then waits on
// `release`; the "fast" mist's lookup does not proceed until it observes
// `entered`. That ordering constraint is deliberate -- mist.Registry.AllByTenant
// iterates a Go map (mist/registry.go:132-135), so which of the two mists
// processTenant reaches first is non-deterministic per run.
//
// Under a regressed SERIAL loop this deadlocks in EITHER iteration order:
//   - blocked first: the loop parks on `<-release` inside blocked's body and
//     never reaches fast's tick at all, so fast's own wait on `entered` is
//     never even started -- runOnce simply never returns.
//   - fast first: fast's body runs immediately and blocks on `<-entered`,
//     which nothing can ever signal because blocked's body -- the only thing
//     that closes `entered` -- has not run yet and, serially, cannot run
//     until fast returns. Deadlock.
//
// A one-sided version (assert-fast-emits-under-require.Eventually, as this
// test previously did) only fails serial about half the time -- whichever
// order chance handed it. This version fails serial on every run, which the
// before/after evidence in the task-200 fix report demonstrates.
//
// The bounded `select`s below turn a serial regression into a clean test
// failure (via the assertion after the timeout) rather than hanging the
// suite forever.
func TestMistTick_MonsterTarget_SlowLookupDoesNotBlockSiblingMist(t *testing.T) {
	tt := mkTickTenant()
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	mt := newTestMistTick(t, reg, rec, nil)

	blocked := mkMonsterMist(t)
	fast := mkMonsterMist(t)
	require.NoError(t, reg.Add(tt, blocked))
	require.NoError(t, reg.Add(tt, fast))

	entered := make(chan struct{}) // closed by blocked's lookup the instant it starts
	release := make(chan struct{}) // closed by the test to let blocked's lookup return

	mt.monstersInRect = func(_ context.Context, mm mist.Mist) ([]monster.RestModel, error) {
		if mm.Id() == blocked.Id() {
			close(entered)
			<-release // never returns until the test says so
			return nil, errors.New("atlas-monsters unavailable")
		}
		// fast's lookup must not proceed until blocked's has demonstrably
		// started -- this is what makes serial execution deadlock instead of
		// racing past on luck.
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			return nil, errors.New("timed out waiting for the blocked mist's lookup to start -- fan-out is not concurrent")
		}
		return []monster.RestModel{mkMonsterRest(9001, 500, 300)}, nil
	}

	done := make(chan struct{})
	go func() {
		mt.runOnce(context.Background())
		close(done)
	}()
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	})

	// runOnce must NOT complete yet: the blocked mist is still parked on
	// release. If it does complete, either the handshake never engaged or
	// the fan-out silently dropped a mist -- either way this is a failure,
	// not a race to tolerate.
	select {
	case <-done:
		t.Fatal("runOnce returned before the blocked mist's lookup was released -- fan-out did not isolate latency, or the handshake never engaged (serial regression)")
	case <-time.After(300 * time.Millisecond):
	}

	// The fast mist's APPLY_STATUS must have landed by now -- proof the two
	// mists' lookups actually ran concurrently, not serially: the fast one
	// waited for `entered` and returned, all while the blocked one is still
	// hanging on `release`.
	require.Len(t, rec.Messages(EnvCommandTopicMonster), 1,
		"sibling mist's tick did not emit while the other mist's rect lookup was still blocked")

	close(release)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runOnce did not complete after the blocked mist's lookup was released")
	}
}

// mkMonsterMist builds a player-cast mist anchored at (500,300) with the
// level-1 Poison Mist rectangle from WZ (lt -110,-82 / rb 110,83), i.e.
// absolute bounds (390,218)-(610,383). The caller registers it.
func mkMonsterMist(t *testing.T) mist.Mist {
	t.Helper()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	return mist.NewBuilder(uuid.New(), f).
		SetOwner("CHARACTER", 1001).
		SetKinds(mistKafka.TargetKindMonster, mistKafka.EffectKindDamageOverTime).
		SetSource(2111003, 1).
		SetOrigin(500, 300).
		SetBounds(-110, -82, 110, 83).
		SetDisease("POISON", 0, 4000*time.Millisecond).
		SetDuration(4000 * time.Millisecond).
		SetTickInterval(1000 * time.Millisecond).
		Build()
}

func mkMonsterRest(uniqueId uint32, x, y int16) monster.RestModel {
	return monster.RestModel{Id: strconv.Itoa(int(uniqueId)), X: x, Y: y}
}

// TestMistTick_MonsterTarget_EmitsApplyStatusPerMonster asserts one
// APPLY_STATUS per monster returned by the rect endpoint, with the exact body
// atlas-monsters' consumer expects (task-200 FR-4.2).
func TestMistTick_MonsterTarget_EmitsApplyStatusPerMonster(t *testing.T) {
	tt := mkTickTenant()
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	mt := newTestMistTick(t, reg, rec, func(context.Context, uint32) (int16, int16, uint16, error) {
		t.Fatal("character position lookup must not run for a MONSTER mist")
		return 0, 0, 0, nil
	})

	m := mkMonsterMist(t)
	require.NoError(t, reg.Add(tt, m))

	var gotRect [4]int16
	mt.monstersInRect = func(_ context.Context, mm mist.Mist) ([]monster.RestModel, error) {
		x1, y1, x2, y2 := mm.Rect()
		gotRect = [4]int16{x1, y1, x2, y2}
		return []monster.RestModel{mkMonsterRest(9001, 500, 300), mkMonsterRest(9002, 610, 383)}, nil
	}

	mt.runOnce(context.Background())

	require.Equal(t, [4]int16{390, 218, 610, 383}, gotRect)

	msgs := rec.Messages(EnvCommandTopicMonster)
	require.Len(t, msgs, 2)
	require.Empty(t, rec.Messages(EnvCommandTopicCharacterBuff))

	var cmd struct {
		MonsterId uint32 `json:"monsterId"`
		Type      string `json:"type"`
		Body      struct {
			SourceType        string           `json:"sourceType"`
			SourceCharacterId uint32           `json:"sourceCharacterId"`
			SourceSkillId     uint32           `json:"sourceSkillId"`
			SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
			Statuses          map[string]int32 `json:"statuses"`
			Duration          uint32           `json:"duration"`
			TickInterval      uint32           `json:"tickInterval"`
		} `json:"body"`
	}
	require.NoError(t, json.Unmarshal(msgs[0].Value, &cmd))
	require.Equal(t, uint32(9001), cmd.MonsterId)
	require.Equal(t, "APPLY_STATUS", cmd.Type)
	require.Equal(t, "PLAYER_SKILL", cmd.Body.SourceType)
	require.Equal(t, uint32(1001), cmd.Body.SourceCharacterId)
	require.Equal(t, uint32(2111003), cmd.Body.SourceSkillId)
	require.Equal(t, uint32(1), cmd.Body.SourceSkillLevel)
	require.Equal(t, map[string]int32{"POISON": 0}, cmd.Body.Statuses)
	require.Equal(t, uint32(4000), cmd.Body.Duration)
	require.Equal(t, uint32(1000), cmd.Body.TickInterval)
}

// TestMistTick_MonsterTarget_BodyKeySetMatchesChannel pins the JSON key set
// against atlas-channel's ApplyStatusCommandBody. COMMAND_TOPIC_MONSTER is a
// shared topic: every registered handler unmarshals every message, so an added
// or renamed key causes decode errors on unrelated handlers.
func TestMistTick_MonsterTarget_BodyKeySetMatchesChannel(t *testing.T) {
	tt := mkTickTenant()
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	mt := newTestMistTick(t, reg, rec, nil)

	m := mkMonsterMist(t)
	require.NoError(t, reg.Add(tt, m))
	mt.monstersInRect = func(context.Context, mist.Mist) ([]monster.RestModel, error) {
		return []monster.RestModel{mkMonsterRest(9001, 500, 300)}, nil
	}
	mt.runOnce(context.Background())

	var envelope map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Messages(EnvCommandTopicMonster)[0].Value, &envelope))
	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(envelope["body"], &body))

	keys := make([]string, 0, len(body))
	for k := range body {
		keys = append(keys, k)
	}
	require.ElementsMatch(t, []string{
		"sourceType", "sourceCharacterId", "sourceSkillId",
		"sourceSkillLevel", "statuses", "duration", "tickInterval",
	}, keys)
}

// TestMistTick_MonsterTarget_DotTickIntervalStrictlyLessThanReapplyInterval
// guards the cross-service cadence relationship that the previous fix wave
// silently broke: monsterDotTickIntervalMs must be strictly less than the
// mist's own re-apply period (m.TickInterval(), driven here by
// SetTickInterval in mkMonsterMist). If a future change re-echoes the mist's
// re-apply interval into the emitted APPLY_STATUS body -- as the earlier,
// incorrect fix attempt did -- the eligible damage window in atlas-monsters
// (P - T, see monsterDotTickIntervalMs's doc comment) collapses to zero and
// no poison damage ever lands, even though every other assertion in this
// file still passes.
func TestMistTick_MonsterTarget_DotTickIntervalStrictlyLessThanReapplyInterval(t *testing.T) {
	tt := mkTickTenant()
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	mt := newTestMistTick(t, reg, rec, nil)

	// Built with the PRODUCTION re-apply cadence
	// (poisonmist.PlayerMistTickIntervalMs, 3000ms) rather than
	// mkMonsterMist's 1000ms fixture value, so this test actually exercises
	// the real P vs T relationship instead of a coincidental match.
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	m := mist.NewBuilder(uuid.New(), f).
		SetOwner("CHARACTER", 1001).
		SetKinds(mistKafka.TargetKindMonster, mistKafka.EffectKindDamageOverTime).
		SetSource(2111003, 1).
		SetOrigin(500, 300).
		SetBounds(-110, -82, 110, 83).
		SetDisease("POISON", 0, 4000*time.Millisecond).
		SetDuration(4000 * time.Millisecond).
		SetTickInterval(3000 * time.Millisecond).
		Build()
	require.NoError(t, reg.Add(tt, m))
	mt.monstersInRect = func(context.Context, mist.Mist) ([]monster.RestModel, error) {
		return []monster.RestModel{mkMonsterRest(9001, 500, 300)}, nil
	}
	mt.runOnce(context.Background())

	msgs := rec.Messages(EnvCommandTopicMonster)
	require.Len(t, msgs, 1)

	var cmd struct {
		Body struct {
			TickInterval uint32 `json:"tickInterval"`
		} `json:"body"`
	}
	require.NoError(t, json.Unmarshal(msgs[0].Value, &cmd))

	reapplyIntervalMs := uint32(m.TickInterval().Milliseconds())
	require.Less(t, cmd.Body.TickInterval, reapplyIntervalMs,
		"emitted DoT tickInterval must be strictly less than the mist's re-apply interval, or the atlas-monsters damage window (P-T) collapses to zero")
}

// TestMistTick_MonsterTarget_LookupFailureIsolatedPerMist asserts a failing
// rect query on one mist does not prevent another mist in the same tenant from
// ticking (FR-4.6, NFR-2).
func TestMistTick_MonsterTarget_LookupFailureIsolatedPerMist(t *testing.T) {
	tt := mkTickTenant()
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	mt := newTestMistTick(t, reg, rec, nil)

	bad := mkMonsterMist(t)
	good := mkMonsterMist(t)
	require.NoError(t, reg.Add(tt, bad))
	require.NoError(t, reg.Add(tt, good))

	mt.monstersInRect = func(_ context.Context, mm mist.Mist) ([]monster.RestModel, error) {
		if mm.Id() == bad.Id() {
			return nil, errors.New("atlas-monsters unavailable")
		}
		return []monster.RestModel{mkMonsterRest(9001, 500, 300)}, nil
	}

	mt.runOnce(context.Background())
	require.Len(t, rec.Messages(EnvCommandTopicMonster), 1)
}

// TestMistTick_MonsterTarget_NoMonsters_EmitsNothing asserts an empty rect
// result produces no commands and still advances lastTick (so a persistently
// empty map does not spin).
func TestMistTick_MonsterTarget_NoMonsters_EmitsNothing(t *testing.T) {
	tt := mkTickTenant()
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	mt := newTestMistTick(t, reg, rec, nil)

	m := mkMonsterMist(t)
	require.NoError(t, reg.Add(tt, m))
	mt.monstersInRect = func(context.Context, mist.Mist) ([]monster.RestModel, error) {
		return nil, nil
	}

	mt.runOnce(context.Background())
	require.Empty(t, rec.Messages(EnvCommandTopicMonster))

	after := reg.AllByTenant(tt)
	require.Len(t, after, 1)
	require.False(t, after[0].ShouldTick())
}
