package poisonbomb

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/skill/handler/mistcast"
	"context"
	"errors"
	"testing"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// FR-7.1/FR-7.3: the client delivers Poison Bomb on an ATTACK packet, so it
// must be on the attack-cast registry and NOT on the use-skill one --
// registering it there would never fire AND would silently zero its MP cost.
// Registration is by Identity so one call covers every version that binds it.
func TestInit_RegistersOnAttackCastRegistryByIdentity(t *testing.T) {
	_, ok := channelhandler.LookupAttackCast(skill2.NightWalkerStage3PoisonBomb)
	require.True(t, ok, "Poison Bomb must be registered on the attack-cast registry")

	_, wrong := channelhandler.Lookup(skill2.NightWalkerStage3PoisonBomb)
	require.False(t, wrong, "Poison Bomb must NOT be on the use-skill registry")
}

func TestApply_EmitsMonsterDotMistWithWireSkillId(t *testing.T) {
	l, _ := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody

	origLoad, origEmit := loadCaster, emitCreate
	t.Cleanup(func() { loadCaster, emitCreate = origLoad, origEmit })
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
		return 300, 120, nil
	}
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}

	e := stubEffect(40000, -100, -82, 100, 83)
	f := field.NewBuilder(0, 0, 100000000).Build()

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, skill2.Id(14111006), 30, e, nil))

	require.Len(t, emitted, 1)
	b := emitted[0]
	require.Equal(t, mistmsg.TargetKindMonster, b.TargetKind)
	require.Equal(t, mistmsg.EffectKindDamageOverTime, b.EffectKind)
	require.Equal(t, "POISON", b.Disease)
	// Target-derived: atlas-monsters resolves and overwrites it (FR-6.2).
	require.Equal(t, int32(0), b.DiseaseValue)
	require.Equal(t, mistcast.PlayerMistTickIntervalMs, b.TickIntervalMs)
	// FR-7.4: the WIRE id, not the Identity.
	require.Equal(t, uint32(14111006), b.SourceSkillId)
	require.Equal(t, int16(300), b.OriginX)
	require.Equal(t, int32(0), b.RecoveryMp)
	require.Nil(t, b.PartyMemberIds)
}

// The shortest legitimate lifetime clears the sub-tick gate; pinned so a
// cadence change cannot silently disable the skill at low levels.
func TestApply_ShortestRealLifetimeIsAccepted(t *testing.T) {
	l, _ := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody

	origLoad, origEmit := loadCaster, emitCreate
	t.Cleanup(func() { loadCaster, emitCreate = origLoad, origEmit })
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) { return 0, 0, nil }
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}

	e := stubEffect(4000, -100, -82, 100, 83)
	f := field.NewBuilder(0, 0, 100000000).Build()

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, skill2.Id(14111006), 1, e, nil))
	require.Len(t, emitted, 1)
}

// stubEffect hydrates an effect.Model through the supported construction
// path -- effect.Model has unexported fields and no exported builder.
func stubEffect(duration int32, ltX, ltY, rbX, rbY int16) effect.Model {
	m, err := effect.Extract(effect.RestModel{
		Duration: duration,
		LT:       &effect.PointRestModel{X: ltX, Y: ltY},
		RB:       &effect.PointRestModel{X: rbX, Y: rbY},
	})
	if err != nil {
		panic(err)
	}
	return m
}

// TestApply_AnchorsMistAtGrenadeLandingPoint is the regression for the
// task-218 field report: the mist appeared at the caster's feet regardless of
// how far the bomb was thrown.
//
// Poison Bomb is a keydown skill — the longer the key is held, the further the
// throw — and the client sends the landing point on the attack packet
// (AttackInfo.GrenadeX/GrenadeY) after drawing the explosion there. The cast
// must anchor the cloud on that point, NOT on the caster, or the DoT rectangle
// sits a whole throw-distance away from what the player sees.
//
// loadCaster is deliberately rigged to fail: with a packet-supplied origin
// there is nothing to look up, and a cast whose anchor the client already
// fixed must not be sunk by an unrelated character-service error.
func TestApply_AnchorsMistAtGrenadeLandingPoint(t *testing.T) {
	l, _ := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody

	origLoad, origEmit := loadCaster, emitCreate
	t.Cleanup(func() { loadCaster, emitCreate = origLoad, origEmit })
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
		return 300, 120, errors.New("character service unavailable")
	}
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}

	e := stubEffect(40000, -100, -82, 100, 83)
	f := field.NewBuilder(0, 0, 100000000).Build()
	landing := point.NewModel(point.X(-540), point.Y(421))

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, skill2.Id(14111006), 30, e, &landing))

	require.Len(t, emitted, 1)
	require.EqualValues(t, -540, emitted[0].OriginX, "mist must be anchored where the bomb landed, not at the caster")
	require.EqualValues(t, 421, emitted[0].OriginY, "mist must be anchored where the bomb landed, not at the caster")
}

// TestApply_FallsBackToCasterWithoutGrenadeOrigin pins the other half of the
// contract: a nil origin (no grenade block on the packet) still plants the
// mist at the caster, so the fix cannot regress the non-thrown path.
func TestApply_FallsBackToCasterWithoutGrenadeOrigin(t *testing.T) {
	l, _ := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody

	origLoad, origEmit := loadCaster, emitCreate
	t.Cleanup(func() { loadCaster, emitCreate = origLoad, origEmit })
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
		return 300, 120, nil
	}
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}

	e := stubEffect(40000, -100, -82, 100, 83)
	f := field.NewBuilder(0, 0, 100000000).Build()

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, skill2.Id(14111006), 30, e, nil))

	require.Len(t, emitted, 1)
	require.EqualValues(t, 300, emitted[0].OriginX)
	require.EqualValues(t, 120, emitted[0].OriginY)
}
