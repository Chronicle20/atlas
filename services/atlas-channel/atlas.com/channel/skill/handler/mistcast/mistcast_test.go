package mistcast

import (
	"atlas-channel/data/skill/effect"
	"context"
	"errors"
	"testing"

	mistmsg "atlas-channel/kafka/message/mist"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// stubEffect builds an effect.Model with only the fields a mist cast reads.
// effect.Model has unexported fields and no exported constructor; hydrating
// through effect.Extract on a RestModel literal is the supported
// construction path (same helper shape as poisonmist_test.go:38-47).
func stubEffect(duration int32, ltX, ltY, rbX, rbY int16) effect.Model {
	return stubEffectX(duration, 0, ltX, ltY, rbX, rbY)
}

// stubEffectX additionally sets the `x` node, which Recovery Aura reads as
// its per-tick MP magnitude.
func stubEffectX(duration int32, x int16, ltX, ltY, rbX, rbY int16) effect.Model {
	m, err := effect.Extract(effect.RestModel{
		Duration: duration,
		X:        x,
		LT:       &effect.PointRestModel{X: ltX, Y: ltY},
		RB:       &effect.PointRestModel{X: rbX, Y: rbY},
	})
	if err != nil {
		panic(err)
	}
	return m
}

func dotParams() Params {
	return Params{
		SkillName:  "Test Mist",
		TargetKind: mistmsg.TargetKindMonster,
		EffectKind: mistmsg.EffectKindDamageOverTime,
		Disease:    "POISON",
		TickMs:     PlayerMistTickIntervalMs,
	}
}

// run drives Cast with recording seams and returns everything emitted plus
// the log hook, so each case asserts on the wire body and the warning.
func run(t *testing.T, e effect.Model, p Params) ([]mistmsg.CreateCommandBody, *test.Hook) {
	t.Helper()
	l, hook := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody
	s := Seams{
		LoadCaster: func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
			return 250, -80, nil
		},
		EmitCreate: func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
			emitted = append(emitted, body)
			return nil
		},
	}
	f := field.NewBuilder(0, 0, 100000000).Build()
	require.NoError(t, Cast(l, context.Background(), f, 1001, 2111003, 20, e, p, s))
	return emitted, hook
}

func TestCast_HappyPath_EmitsExactlyOneCreate(t *testing.T) {
	emitted, _ := run(t, stubEffect(40000, -110, -82, 110, 83), dotParams())

	require.Len(t, emitted, 1)
	b := emitted[0]
	require.Equal(t, "CHARACTER", b.OwnerType)
	require.Equal(t, uint32(1001), b.OwnerId)
	require.Equal(t, int16(250), b.OriginX)
	require.Equal(t, int16(-80), b.OriginY)
	require.Equal(t, int16(-110), b.LtX)
	require.Equal(t, int16(83), b.RbY)
	require.Equal(t, int64(40000), b.Duration)
	require.Equal(t, int64(40000), b.DiseaseDuration)
	require.Equal(t, PlayerMistTickIntervalMs, b.TickIntervalMs)
	require.Equal(t, mistmsg.TargetKindMonster, b.TargetKind)
	require.Equal(t, mistmsg.EffectKindDamageOverTime, b.EffectKind)
	require.Equal(t, "POISON", b.Disease)
	// Target-derived magnitude: atlas-monsters resolves and overwrites it.
	require.Equal(t, int32(0), b.DiseaseValue)
	// The WIRE id, not the Identity: the client picks its rendering arm from it.
	require.Equal(t, uint32(2111003), b.SourceSkillId)
	require.Equal(t, uint32(20), b.SourceSkillLevel)
}

func TestCast_ZeroLifetime_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(0, -110, -82, 110, 83), dotParams())
	require.Empty(t, emitted)
	require.Contains(t, hook.LastEntry().Message, "no lifetime")
	require.Contains(t, hook.LastEntry().Message, "Test Mist")
}

func TestCast_LifetimeShorterThanOneTick_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(int32(PlayerMistTickIntervalMs)-1, -110, -82, 110, 83), dotParams())
	require.Empty(t, emitted)
	require.Contains(t, hook.LastEntry().Message, "shorter than one tick")
}

func TestCast_LifetimeEqualToOneTick_Accepted(t *testing.T) {
	emitted, _ := run(t, stubEffect(int32(PlayerMistTickIntervalMs), -110, -82, 110, 83), dotParams())
	require.Len(t, emitted, 1)
}

// A PROTECTION mist never ticks, so the sub-tick gate must not apply to it.
// Smokescreen's real lifetime (31s) clears it anyway; this pins the rule.
func TestCast_ZeroTickInterval_SkipsSubTickGate(t *testing.T) {
	p := Params{
		SkillName:  "Smoke Test",
		TargetKind: mistmsg.TargetKindCharacter,
		EffectKind: mistmsg.EffectKindProtection,
		TickMs:     0,
	}
	emitted, _ := run(t, stubEffect(1000, -110, -82, 110, 83), p)
	require.Len(t, emitted, 1)
	require.Equal(t, int64(0), emitted[0].TickIntervalMs)
	require.Equal(t, "", emitted[0].Disease)
}

func TestCast_DegenerateRectangle_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(40000, 110, -82, 110, 83), dotParams())
	require.Empty(t, emitted)
	require.Contains(t, hook.LastEntry().Message, "degenerate rectangle")
}

// FR-8.2: reject, never truncate. The client computes its own tEnd from its
// own WZ, so a server-side clamp desynchronises it.
func TestCast_ImplausibleLifetime_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(MaxPlayerMistDurationMs+1, -110, -82, 110, 83), dotParams())
	require.Empty(t, emitted)
	require.Contains(t, hook.LastEntry().Message, "implausible lifetime")
}

func TestCast_CasterLoadFailure_EmitsNothingAndReturnsNil(t *testing.T) {
	l, hook := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody
	s := Seams{
		LoadCaster: func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
			return 0, 0, errors.New("boom")
		},
		EmitCreate: func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
			emitted = append(emitted, body)
			return nil
		},
	}
	f := field.NewBuilder(0, 0, 100000000).Build()

	require.NoError(t, Cast(l, context.Background(), f, 1001, 2111003, 20, stubEffect(40000, -110, -82, 110, 83), dotParams(), s))
	require.Empty(t, emitted)
	require.Contains(t, hook.LastEntry().Message, "failed to load caster")
}

func TestCast_EmitFailure_ReturnsNil(t *testing.T) {
	l, hook := test.NewNullLogger()
	s := Seams{
		LoadCaster: func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
			return 0, 0, nil
		},
		EmitCreate: func(logrus.FieldLogger, context.Context, mistmsg.CreateCommandBody) error {
			return errors.New("kafka down")
		},
	}
	f := field.NewBuilder(0, 0, 100000000).Build()

	require.NoError(t, Cast(l, context.Background(), f, 1001, 2111003, 20, stubEffect(40000, -110, -82, 110, 83), dotParams(), s))
	require.Contains(t, hook.LastEntry().Message, "failed to emit CREATE")
}

// FR-5: a RECOVERY cast carries its magnitude and party snapshot explicitly;
// DiseaseValue is never overloaded for it.
func TestCast_RecoveryParams_CarryMagnitudeAndParty(t *testing.T) {
	p := Params{
		SkillName:      "Recovery Test",
		TargetKind:     mistmsg.TargetKindCharacter,
		EffectKind:     mistmsg.EffectKindRecovery,
		TickMs:         PlayerMistTickIntervalMs,
		RecoveryMp:     38,
		PartyMemberIds: []uint32{1001, 1002},
	}
	emitted, _ := run(t, stubEffect(30000, -200, -125, 200, 30), p)

	require.Len(t, emitted, 1)
	require.Equal(t, int32(38), emitted[0].RecoveryMp)
	require.Equal(t, []uint32{1001, 1002}, emitted[0].PartyMemberIds)
	require.Equal(t, int32(0), emitted[0].DiseaseValue)
	require.Equal(t, "", emitted[0].Disease)
}

// FR-6.4: the re-apply period P must strictly exceed the DoT tick interval T
// that atlas-maps emits, or the eligible damage window is exactly zero and
// the mist deals no damage at any tuning. T is atlas-maps'
// monsterDotTickIntervalMs; it is duplicated here as a literal on purpose --
// the two services share no module, and this test is what catches a change
// to either side.
func TestPlayerMistTickInterval_LeavesANonZeroDamageWindow(t *testing.T) {
	const monsterDotTickIntervalMs int64 = 1000 // atlas-maps tasks/mist_tick.go

	require.Greater(t, PlayerMistTickIntervalMs, monsterDotTickIntervalMs)
	require.Equal(t, int64(2000), PlayerMistTickIntervalMs-monsterDotTickIntervalMs)
}
