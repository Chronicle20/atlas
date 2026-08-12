package recoveryaura

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
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func TestInit_RegistersOnUseSkillRegistryByIdentity(t *testing.T) {
	_, ok := channelhandler.Lookup(skill2.EvanStage8RecoveryAura)
	require.True(t, ok, "Recovery Aura must be registered on the use-skill registry")

	_, wrong := channelhandler.LookupAttackCast(skill2.EvanStage8RecoveryAura)
	require.False(t, wrong, "Recovery Aura must NOT be on the attack-cast registry")
}

// run drives Apply with recording seams and a stubbed party snapshot.
func run(t *testing.T, x int16, party []uint32) []mistmsg.CreateCommandBody {
	t.Helper()
	l, _ := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody

	origLoad, origEmit, origParty := loadCaster, emitCreate, loadPartyMemberIds
	t.Cleanup(func() { loadCaster, emitCreate, loadPartyMemberIds = origLoad, origEmit, origParty })
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) { return 10, 20, nil }
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}
	loadPartyMemberIds = func(logrus.FieldLogger, context.Context, uint32) []uint32 { return party }

	e := stubEffect(30000, x)
	f := field.NewBuilder(0, 0, 100000000).Build()
	info := packetmodel.NewSkillUsageInfoBuilder().SetSkillId(22161003).SetSkillLevel(1).Build()

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, info, e))
	return emitted
}

// stubEffect hydrates an effect.Model with Recovery Aura's real rectangle
// (lt(-200,-125) rb(200,30), identical on every version that binds it) and
// the given `x` magnitude. effect.Model has unexported fields and no
// exported builder; Extract is the supported construction path.
func stubEffect(duration int32, x int16) effect.Model {
	m, err := effect.Extract(effect.RestModel{
		Duration: duration,
		X:        x,
		LT:       &effect.PointRestModel{X: -200, Y: -125},
		RB:       &effect.PointRestModel{X: 200, Y: 30},
	})
	if err != nil {
		panic(err)
	}
	return m
}

// FR-1.2/FR-5.1: the per-tick magnitude is the WZ `x` node (38 at L1, 80 at
// L15), not a constant, and it travels in RecoveryMp -- never overloaded onto
// DiseaseValue.
func TestApply_MagnitudeIsTheWzXNode(t *testing.T) {
	emitted := run(t, 38, []uint32{1001, 1002})

	require.Len(t, emitted, 1)
	b := emitted[0]
	require.Equal(t, mistmsg.TargetKindCharacter, b.TargetKind)
	require.Equal(t, mistmsg.EffectKindRecovery, b.EffectKind)
	require.Equal(t, int32(38), b.RecoveryMp)
	require.Equal(t, int32(0), b.DiseaseValue)
	require.Equal(t, "", b.Disease)
	require.Equal(t, mistcast.PlayerMistTickIntervalMs, b.TickIntervalMs)
	require.Equal(t, uint32(22161003), b.SourceSkillId)
}

// FR-5.2: the party snapshot travels on the command; atlas-maps has no party
// client and heals only ids in this set.
func TestApply_CarriesThePartySnapshot(t *testing.T) {
	emitted := run(t, 38, []uint32{1001, 1002, 1003})
	require.Equal(t, []uint32{1001, 1002, 1003}, emitted[0].PartyMemberIds)
}

// A magnitude of 0 would create a mist that heals nothing every 3s for 30s.
// Reject it rather than emit a no-op cloud.
func TestApply_ZeroMagnitude_Rejected(t *testing.T) {
	l, hook := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody

	origLoad, origEmit, origParty := loadCaster, emitCreate, loadPartyMemberIds
	t.Cleanup(func() { loadCaster, emitCreate, loadPartyMemberIds = origLoad, origEmit, origParty })
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) { return 0, 0, nil }
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}
	loadPartyMemberIds = func(logrus.FieldLogger, context.Context, uint32) []uint32 { return []uint32{1001} }

	e := stubEffect(30000, 0)
	f := field.NewBuilder(0, 0, 100000000).Build()
	info := packetmodel.NewSkillUsageInfoBuilder().SetSkillId(22161003).SetSkillLevel(1).Build()

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, info, e))
	require.Empty(t, emitted)
	require.Contains(t, hook.LastEntry().Message, "no recovery magnitude")
}

// A soloing caster still gets their own aura: the snapshot always contains
// the caster, even when the party lookup returns nothing.
func TestPartyMemberIdsOrSelf_AlwaysIncludesCaster(t *testing.T) {
	require.Equal(t, []uint32{1001}, withCaster(nil, 1001))
	require.Equal(t, []uint32{1002, 1001}, withCaster([]uint32{1002}, 1001))
	require.Equal(t, []uint32{1001, 1002}, withCaster([]uint32{1001, 1002}, 1001))
}
