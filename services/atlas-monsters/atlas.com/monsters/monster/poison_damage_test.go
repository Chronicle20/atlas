package monster

import (
	"atlas-monsters/monster/information"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// beetleMaxHp is the live max HP of monster 7130002 (Beetle) as served by
// atlas-data for GMS 83.1. Used so the expected magnitudes below are real
// numbers rather than round ones.
const beetleMaxHp uint32 = 15200

func TestResolvePoisonDamage(t *testing.T) {
	tests := []struct {
		name       string
		maxHp      uint32
		skillLevel uint32
		want       int32
	}{
		// 15200/40 divides evenly.
		{name: "beetle at max level", maxHp: beetleMaxHp, skillLevel: 30, want: 380},
		// 15200/69 = 220.28..., which must round UP, not truncate to 220.
		{name: "beetle at level 1 rounds up", maxHp: beetleMaxHp, skillLevel: 1, want: 221},
		{name: "zero level uses divisor 70", maxHp: 14000, skillLevel: 0, want: 200},
		// The magnitude travels to the client in a signed 16-bit field.
		{name: "capped at signed 16-bit max", maxHp: 4_000_000, skillLevel: 30, want: MaxPoisonDamage},
		// Divisor must never reach zero or go negative.
		{name: "level at the divisor boundary", maxHp: 500, skillLevel: 70, want: 500},
		{name: "level past the divisor boundary", maxHp: 500, skillLevel: 99, want: 500},
		{name: "zero max hp", maxHp: 0, skillLevel: 30, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ResolvePoisonDamage(tt.maxHp, tt.skillLevel))
		})
	}
}

func TestStatusEffect_WithStatus_DoesNotMutateSource(t *testing.T) {
	statuses := map[string]int32{StatusPoison: 0}
	se := NewStatusEffect(SourceTypePlayerSkill, 1, 2111003, 30, statuses, 40*time.Second, time.Second)

	updated := se.WithStatus(StatusPoison, 380)

	require.Equal(t, int32(380), updated.Statuses()[StatusPoison])
	require.Equal(t, int32(0), se.Statuses()[StatusPoison], "receiver must be unchanged")
	require.Equal(t, int32(0), statuses[StatusPoison], "caller's map must be unchanged")
}

// newPoisonTestProcessor builds a processor over the in-memory registry with
// emission stubbed out, plus a monster to poison.
func newPoisonTestProcessor(t *testing.T, hp uint32) (*ProcessorImpl, tenant.Model, Model) {
	t.Helper()
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	// Stub the monster-info lookup: a plain non-boss with no resistances, so
	// the immunity gates pass and the test does not attempt a REST call.
	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().Build(), nil
	}
	t.Cleanup(func() { testInformationLookup = prevHook })

	m := r.CreateMonster(ctx, tm, testField(), 7130002, 0, 0, 0, 0, 0, hp, 120, "", "")

	p := &ProcessorImpl{
		l:    logrus.New(),
		ctx:  ctx,
		t:    tm,
		emit: func(_ string, _ model.Provider[[]kafka.Message]) error { return nil },
		inFieldFn: func(_ field.Model) ([]uint32, error) {
			return nil, nil
		},
	}
	return p, tm, m
}

// A player-cast mist sends POISON with magnitude 0 because the magnitude
// depends on the TARGET's max HP. ApplyStatusEffect must resolve it, so the
// stored effect — and therefore the temporary-stat packet the channel builds
// from the STATUS_APPLIED event — carries the real per-tick damage.
func TestApplyStatusEffect_ResolvesPoisonMagnitude(t *testing.T) {
	p, tm, m := newPoisonTestProcessor(t, beetleMaxHp)

	effect := NewStatusEffect(SourceTypePlayerSkill, 1, 2111003, 30,
		map[string]int32{StatusPoison: 0}, 40*time.Second, time.Second)
	require.NoError(t, p.ApplyStatusEffect(m.UniqueId(), effect))

	stored, err := GetMonsterRegistry().GetMonster(tm, m.UniqueId())
	require.NoError(t, err)
	require.Len(t, stored.StatusEffects(), 1)
	require.Equal(t, int32(380), stored.StatusEffects()[0].Statuses()[StatusPoison],
		"client renders its own tick numbers from this magnitude; 0 renders nothing")
}

// Poison can never land the kill (a tick is capped at currentHP-1), so
// applying it to a monster already at 1 HP can only produce an effect that
// never does anything — and a re-applying mist would churn it every cycle.
func TestApplyStatusEffect_RejectsPoisonAtOneHp(t *testing.T) {
	p, tm, m := newPoisonTestProcessor(t, 1)

	effect := NewStatusEffect(SourceTypePlayerSkill, 1, 2111003, 30,
		map[string]int32{StatusPoison: 0}, 40*time.Second, time.Second)
	require.Error(t, p.ApplyStatusEffect(m.UniqueId(), effect))

	stored, err := GetMonsterRegistry().GetMonster(tm, m.UniqueId())
	require.NoError(t, err)
	require.Empty(t, stored.StatusEffects())
}

// The tick must apply exactly the magnitude that was resolved at apply time,
// not recompute it — otherwise the number the client renders and the damage
// the server applies can drift apart.
func TestCalculatePoisonDamage_UsesResolvedMagnitude(t *testing.T) {
	task := &StatusExpirationTask{l: logrus.New()}
	m := NewMonster(testField(), 1, 7130002, 0, 0, 0, 0, 0, beetleMaxHp, 120, "", "")

	resolved := NewStatusEffect(SourceTypePlayerSkill, 1, 2111003, 30,
		map[string]int32{StatusPoison: 380}, 40*time.Second, time.Second)
	require.Equal(t, uint32(380), task.calculatePoisonDamage(m, resolved))

	// A magnitude the caster could not have known differs from the formula;
	// the stored value still wins.
	pinned := NewStatusEffect(SourceTypePlayerSkill, 1, 2111003, 30,
		map[string]int32{StatusPoison: 42}, 40*time.Second, time.Second)
	require.Equal(t, uint32(42), task.calculatePoisonDamage(m, pinned))

	// Fallback for an effect that never passed through ApplyStatusEffect.
	unresolved := NewStatusEffect(SourceTypePlayerSkill, 1, 2111003, 30,
		map[string]int32{StatusPoison: 0}, 40*time.Second, time.Second)
	require.Equal(t, uint32(380), task.calculatePoisonDamage(m, unresolved))
}

// The DAMAGED event must report the damage THIS event applied. DamageEntries
// carries the monster's running per-character totals and answers a different
// question; reading its last element as "the damage" reports a cumulative
// figure that grows with every hit.
func TestDamagedStatusEventProvider_DamageIsNotCumulative(t *testing.T) {
	m := NewMonster(testField(), 1, 7130002, 0, 0, 0, 0, 0, beetleMaxHp, 120, "", "")
	summary := []entry{{CharacterId: 1, Damage: 10_800, LastHitMs: 1}}

	msgs, err := damagedStatusEventProvider(m, 1, 1, false, DamageSourceDamageOverTime, 380, summary)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var got struct {
		Type string                 `json:"type"`
		Body statusEventDamagedBody `json:"body"`
	}
	require.NoError(t, json.Unmarshal(msgs[0].Value, &got))
	require.Equal(t, EventMonsterStatusDamaged, got.Type)
	require.Equal(t, uint32(380), got.Body.Damage)
	require.Equal(t, uint32(10_800), got.Body.DamageEntries[0].Damage,
		"cumulative total still travels, for kill credit")
}
