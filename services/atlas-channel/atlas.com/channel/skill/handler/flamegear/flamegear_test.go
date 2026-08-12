package flamegear

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/skill/handler/mistcast"
	"context"
	"testing"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// FR-7.1/FR-7.3: the client delivers Flame Gear on an ATTACK packet, so it
// must be on the attack-cast registry and NOT on the use-skill one --
// registering it there would never fire AND would silently zero its MP cost.
// Registration is by Identity so one call covers every version that binds it.
func TestInit_RegistersOnAttackCastRegistryByIdentity(t *testing.T) {
	_, ok := channelhandler.LookupAttackCast(skill2.BlazeWizardStage3FlameGear)
	require.True(t, ok, "Flame Gear must be registered on the attack-cast registry")

	_, wrong := channelhandler.Lookup(skill2.BlazeWizardStage3FlameGear)
	require.False(t, wrong, "Flame Gear must NOT be on the use-skill registry")
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

	e := stubEffect(40000, -200, -250, 200, 30)
	f := field.NewBuilder(0, 0, 100000000).Build()

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, skill2.Id(12111005), 30, e))

	require.Len(t, emitted, 1)
	b := emitted[0]
	require.Equal(t, mistmsg.TargetKindMonster, b.TargetKind)
	require.Equal(t, mistmsg.EffectKindDamageOverTime, b.EffectKind)
	require.Equal(t, "POISON", b.Disease)
	// Target-derived: atlas-monsters resolves and overwrites it (FR-6.3).
	require.Equal(t, int32(0), b.DiseaseValue)
	require.Equal(t, mistcast.PlayerMistTickIntervalMs, b.TickIntervalMs)
	// FR-7.4: the WIRE id, not the Identity.
	require.Equal(t, uint32(12111005), b.SourceSkillId)
	require.Equal(t, int16(300), b.OriginX)
	require.Equal(t, int32(0), b.RecoveryMp)
	require.Nil(t, b.PartyMemberIds)
}

// The shortest legitimate lifetime (4000ms at L1 on gms 72-92/jms) clears
// the sub-tick gate; pinned so a cadence change cannot silently disable the
// skill at low levels.
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

	e := stubEffect(4000, -200, -250, 200, 30)
	f := field.NewBuilder(0, 0, 100000000).Build()

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, skill2.Id(12111005), 1, e))
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
