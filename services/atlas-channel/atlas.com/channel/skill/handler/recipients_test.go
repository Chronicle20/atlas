package handler

import (
	"context"
	"errors"
	"io"
	"sort"
	"testing"

	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/party"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

// installPartySeams replaces the three external-lookup seams used by the
// party selectors with deterministic in-memory implementations.
func installPartySeams(t *testing.T, p party.Model, partyErr error, inMap map[uint32]struct{}, members map[uint32]character.Model) {
	t.Helper()
	prevParty := loadCasterPartyFunc
	prevInMap := inMapCharacterIdsFunc
	prevMember := loadPartyMemberFunc

	loadCasterPartyFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (party.Model, error) {
		return p, partyErr
	}
	inMapCharacterIdsFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model) map[uint32]struct{} {
		return inMap
	}
	loadPartyMemberFunc = func(_ logrus.FieldLogger, _ context.Context, memberId uint32) (character.Model, error) {
		mc, ok := members[memberId]
		if !ok {
			return character.Model{}, errors.New("member not found")
		}
		return mc, nil
	}

	t.Cleanup(func() {
		loadCasterPartyFunc = prevParty
		inMapCharacterIdsFunc = prevInMap
		loadPartyMemberFunc = prevMember
	})
}

func mkPartyMember(id uint32, online bool, ch channel.Id, mapId _map.Id) party.MemberModel {
	m, _ := party.ExtractMember(party.MemberRestModel{
		Id:        id,
		Name:      "m",
		Level:     10,
		WorldId:   world.Id(0),
		ChannelId: ch,
		MapId:     mapId,
		Instance:  uuid.Nil,
		Online:    online,
	})
	return m
}

func mkMemberChar(id uint32, hp uint16) character.Model {
	return character.NewModelBuilder().SetId(id).SetHp(hp).SetMaxHp(1000).MustBuild()
}

func recipientIds(rs []PartyRecipient) []uint32 {
	out := make([]uint32, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.Id())
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func eqIds(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

const (
	testCasterId = uint32(100)
	testMemberA  = uint32(101)
	testMemberB  = uint32(102)
)

// threePersonParty builds a party of [caster, A, B] all online and in the
// caster's channel/map by default. Bit i of the affected-member bitmap maps
// to member index i in p.Members(): caster=0, A=1, B=2.
func threePersonParty(a, b party.MemberModel) party.Model {
	return party.NewBuilder().
		SetId(1).
		SetLeaderId(testCasterId).
		SetMembers([]party.MemberModel{
			mkPartyMember(testCasterId, true, channel.Id(0), _map.Id(40000)),
			a,
			b,
		}).
		Build()
}

// Bug-2 regression: a party buff has no LT/RB rectangle in its WZ effect.
// The old SelectInRangePartyMembers would short-circuit to caster-only and
// buff nobody; SelectPartyMembersInMap must return all bitmap-selected,
// in-map members.
func TestSelectPartyMembersInMap_AppliesMapWideWithoutRectangle(t *testing.T) {
	a := mkPartyMember(testMemberA, true, channel.Id(0), _map.Id(40000))
	b := mkPartyMember(testMemberB, true, channel.Id(0), _map.Id(40000))
	installPartySeams(t, threePersonParty(a, b), nil,
		map[uint32]struct{}{testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			testMemberA: mkMemberChar(testMemberA, 500),
			testMemberB: mkMemberChar(testMemberB, 500),
		},
	)

	got := recipientIds(SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b110))
	if want := []uint32{testMemberA, testMemberB}; !eqIds(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSelectPartyMembersInMap_BitmapMasksOutMember(t *testing.T) {
	a := mkPartyMember(testMemberA, true, channel.Id(0), _map.Id(40000))
	b := mkPartyMember(testMemberB, true, channel.Id(0), _map.Id(40000))
	installPartySeams(t, threePersonParty(a, b), nil,
		map[uint32]struct{}{testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			testMemberA: mkMemberChar(testMemberA, 500),
			testMemberB: mkMemberChar(testMemberB, 500),
		},
	)

	// Only bit 1 (member A) set.
	got := recipientIds(SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b010))
	if want := []uint32{testMemberA}; !eqIds(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSelectPartyMembersInMap_ExcludesDifferentMap(t *testing.T) {
	a := mkPartyMember(testMemberA, true, channel.Id(0), _map.Id(40000))
	b := mkPartyMember(testMemberB, true, channel.Id(0), _map.Id(99999)) // different map
	installPartySeams(t, threePersonParty(a, b), nil,
		map[uint32]struct{}{testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			testMemberA: mkMemberChar(testMemberA, 500),
			testMemberB: mkMemberChar(testMemberB, 500),
		},
	)

	got := recipientIds(SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b110))
	if want := []uint32{testMemberA}; !eqIds(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSelectPartyMembersInMap_ExcludesOffline(t *testing.T) {
	a := mkPartyMember(testMemberA, false, channel.Id(0), _map.Id(40000)) // offline
	b := mkPartyMember(testMemberB, true, channel.Id(0), _map.Id(40000))
	installPartySeams(t, threePersonParty(a, b), nil,
		map[uint32]struct{}{testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			testMemberA: mkMemberChar(testMemberA, 500),
			testMemberB: mkMemberChar(testMemberB, 500),
		},
	)

	got := recipientIds(SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b110))
	if want := []uint32{testMemberB}; !eqIds(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSelectPartyMembersInMap_ExcludesNoLiveSession(t *testing.T) {
	a := mkPartyMember(testMemberA, true, channel.Id(0), _map.Id(40000))
	b := mkPartyMember(testMemberB, true, channel.Id(0), _map.Id(40000))
	installPartySeams(t, threePersonParty(a, b), nil,
		map[uint32]struct{}{testMemberA: {}}, // B has no live session
		map[uint32]character.Model{
			testMemberA: mkMemberChar(testMemberA, 500),
			testMemberB: mkMemberChar(testMemberB, 500),
		},
	)

	got := recipientIds(SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b110))
	if want := []uint32{testMemberA}; !eqIds(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSelectPartyMembersInMap_ExcludesDeadMember(t *testing.T) {
	a := mkPartyMember(testMemberA, true, channel.Id(0), _map.Id(40000))
	b := mkPartyMember(testMemberB, true, channel.Id(0), _map.Id(40000))
	installPartySeams(t, threePersonParty(a, b), nil,
		map[uint32]struct{}{testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			testMemberA: mkMemberChar(testMemberA, 500),
			testMemberB: mkMemberChar(testMemberB, 0), // dead
		},
	)

	got := recipientIds(SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b110))
	if want := []uint32{testMemberA}; !eqIds(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSelectPartyMembersInMap_PartyLoadErrorYieldsEmpty(t *testing.T) {
	installPartySeams(t, party.Model{}, errors.New("party service down"),
		map[uint32]struct{}{},
		map[uint32]character.Model{},
	)

	got := SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b110)
	if len(got) != 0 {
		t.Fatalf("got %d recipients, want 0", len(got))
	}
}

// Heal's selector must preserve the caster-only fallback when the effect
// carries no LT/RB rectangle.
func TestSelectInRangePartyMembers_NoRectangleReturnsNil(t *testing.T) {
	a := mkPartyMember(testMemberA, true, channel.Id(0), _map.Id(40000))
	b := mkPartyMember(testMemberB, true, channel.Id(0), _map.Id(40000))
	installPartySeams(t, threePersonParty(a, b), nil,
		map[uint32]struct{}{testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			testMemberA: mkMemberChar(testMemberA, 500),
			testMemberB: mkMemberChar(testMemberB, 500),
		},
	)

	got := SelectInRangePartyMembers(testLogger(), context.Background(), mkField(), testCasterId, 0, 0, effect.Model{}, 0b110)
	if got != nil {
		t.Fatalf("got %v, want nil (caster-only fallback)", got)
	}
}
