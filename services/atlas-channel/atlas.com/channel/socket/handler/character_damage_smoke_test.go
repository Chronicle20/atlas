package handler

import (
	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	"atlas-channel/mist"
	"atlas-channel/monster"
	"testing"
	"time"

	monsterdata "atlas-channel/data/monster"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func smokeAt(f field.Model, ownerId uint32) mist.Protection {
	return mist.NewProtectionBuilder(uuid.New(), f).
		SetOwnerId(ownerId).
		SetRect(0, 0, 200, 200).
		SetExpiresAt(time.Now().Add(time.Minute)).
		Build()
}

// FR-4.6: the client accepts a smoke area only if its owner is the local
// character or one of their ONLINE party members
// (CAffectedAreaPool::IsSmokeAreaByPoint, v95 @0x434f40). The server must
// match, or a non-party player renders unharmed while the server kills them.
func TestShieldedBySmoke(t *testing.T) {
	f := field.NewBuilder(0, 0, 100000000).Build()
	party := func() []uint32 { return []uint32{2001, 2002} }
	noParty := func() []uint32 { return nil }

	tests := []struct {
		name        string
		covering    []mist.Protection
		characterId uint32
		partyIds    func() []uint32
		want        bool
	}{
		{"own mist shields the caster", []mist.Protection{smokeAt(f, 1001)}, 1001, noParty, true},
		{"party member's mist shields", []mist.Protection{smokeAt(f, 2001)}, 1001, party, true},
		{"non-party mist does not shield", []mist.Protection{smokeAt(f, 3001)}, 1001, party, false},
		{"no covering mist does not shield", nil, 1001, party, false},
		{"one qualifying mist among several is enough", []mist.Protection{smokeAt(f, 3001), smokeAt(f, 2002)}, 1001, party, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, shieldedBySmoke(tc.covering, tc.characterId, tc.partyIds))
		})
	}
}

// The party lookup is a REST call; it must not fire when no mist could
// possibly qualify, i.e. on every ordinary hit.
func TestShieldedBySmoke_DoesNotResolvePartyWhenUnnecessary(t *testing.T) {
	f := field.NewBuilder(0, 0, 100000000).Build()
	calls := 0
	party := func() []uint32 { calls++; return []uint32{2001} }

	require.False(t, shieldedBySmoke(nil, 1001, party))
	require.Equal(t, 0, calls, "no covering mists: party must not be resolved")

	require.True(t, shieldedBySmoke([]mist.Protection{smokeAt(f, 1001)}, 1001, party))
	require.Equal(t, 0, calls, "own mist matches first: party must not be resolved")

	require.False(t, shieldedBySmoke([]mist.Protection{smokeAt(f, 3001)}, 1001, party))
	require.Equal(t, 1, calls, "foreign mist: party resolved exactly once")
}

// FR-4.1/FR-4.5: a shielded character takes zero HP damage, and NOTHING else
// on the mitigation chain runs -- no reflect, no meso spend, no MP loss --
// matching the client's own short-circuit before Power Guard/Meso Guard.
func TestProcessDamageTaken_ProtectiveMist_ZeroesEverything(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	buffLookups := 0
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})
	deps.getBuffs = func(uint32) ([]buff.Model, error) { buffLookups++; return nil, nil }
	deps.inProtectiveMist = func(field.Model, uint32, int16, int16) bool { return true }

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 500, false)
	c := testCharacter(t, job.Id(100), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	require.Zero(t, len(em.hp))
	require.Zero(t, len(em.mp))
	require.Zero(t, len(em.meso))
	require.Zero(t, len(em.reflects))
	require.Zero(t, buffLookups, "the shield short-circuits before the mitigation chain")
}

// FR-4.3: a character who leaves the rectangle takes full damage. The check
// reads the character's CURRENT position, so the worst-case lag is one
// damage event -- there is no tick quantisation.
func TestProcessDamageTaken_OutsideTheMist_TakesFullDamage(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})
	deps.inProtectiveMist = func(field.Model, uint32, int16, int16) bool { return false }

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 500, false)
	c := testCharacter(t, job.Id(100), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	require.Equal(t, []int16{-500}, em.hp)
}
