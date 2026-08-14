package smokescreen

import (
	"atlas-channel/data/skill/effect"
	"context"
	"testing"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// FR-7.1/FR-7.2: 4221006 has no attack nodes on any of the ten live tenants,
// so the client delivers it on USE_SKILL and it belongs on the use-skill
// registry -- where UseSkill has already charged MP and the cooldown.
func TestInit_RegistersOnUseSkillRegistryByIdentity(t *testing.T) {
	_, ok := channelhandler.Lookup(skill2.ShadowerSmokescreen)
	require.True(t, ok, "Smokescreen must be registered on the use-skill registry")

	_, wrong := channelhandler.LookupAttackCast(skill2.ShadowerSmokescreen)
	require.False(t, wrong, "Smokescreen must NOT be on the attack-cast registry")
}

func TestApply_EmitsProtectionMistThatNeverTicks(t *testing.T) {
	l, _ := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody

	origLoad, origEmit := loadCaster, emitCreate
	t.Cleanup(func() { loadCaster, emitCreate = origLoad, origEmit })
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
		return -40, 200, nil
	}
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}

	e, err := effect.Extract(effect.RestModel{
		Duration: 31000,
		LT:       &effect.PointRestModel{X: -110, Y: -82},
		RB:       &effect.PointRestModel{X: 110, Y: 83},
	})
	require.NoError(t, err)
	f := field.NewBuilder(0, 0, 100000000).Build()
	info := packetmodel.NewSkillUsageInfoBuilder().SetSkillId(4221006).SetSkillLevel(30).Build()

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, info, e))

	require.Len(t, emitted, 1)
	b := emitted[0]
	require.Equal(t, mistmsg.TargetKindCharacter, b.TargetKind)
	require.Equal(t, mistmsg.EffectKindProtection, b.EffectKind)
	// A protection mist has no atlas-maps tick: the shield is evaluated on
	// the channel's damage path. A non-zero interval here would make the
	// tick's PROTECTION arm reachable.
	require.Equal(t, int64(0), b.TickIntervalMs)
	require.Equal(t, "", b.Disease)
	require.Equal(t, int32(0), b.DiseaseValue)
	require.Equal(t, int32(0), b.RecoveryMp)
	// FR-7.4: the WIRE id, not the Identity.
	require.Equal(t, uint32(4221006), b.SourceSkillId)
	require.Equal(t, uint32(30), b.SourceSkillLevel)
	require.Equal(t, int64(31000), b.Duration)
}
