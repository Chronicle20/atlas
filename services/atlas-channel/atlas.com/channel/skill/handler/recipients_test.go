package handler

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/party"
	"context"
	"errors"
	"io"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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
// caster's channel/map by default. The v83 client packs the affected-member
// bitmap MSB-first by party slot (CUserLocal::FindParty), so member index i
// maps to bit (5-i): caster=bit5, A(idx1)=bit4, B(idx2)=bit3.
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

	got := recipientIds(SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b11000))
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

	// Only member A selected (idx1 -> bit4, 0b10000).
	got := recipientIds(SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b10000))
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

	got := recipientIds(SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b11000))
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

	got := recipientIds(SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b11000))
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

	got := recipientIds(SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b11000))
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

	got := recipientIds(SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b11000))
	if want := []uint32{testMemberA}; !eqIds(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSelectPartyMembersInMap_PartyLoadErrorYieldsEmpty(t *testing.T) {
	installPartySeams(t, party.Model{}, errors.New("party service down"),
		map[uint32]struct{}{},
		map[uint32]character.Model{},
	)

	got := SelectPartyMembersInMap(testLogger(), context.Background(), mkField(), testCasterId, 0b11000)
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

	got := SelectInRangePartyMembers(testLogger(), context.Background(), mkField(), testCasterId, 0, 0, effect.Model{}, 0b11000)
	if got != nil {
		t.Fatalf("got %v, want nil (caster-only fallback)", got)
	}
}

// installMapSeams replaces the in-map id set and the per-player loader.
// It does NOT touch loadCasterPartyFunc — use installPartySeams for party tests.
func installMapSeams(t *testing.T, inMap map[uint32]struct{}, players map[uint32]character.Model) {
	t.Helper()
	prevInMap := inMapCharacterIdsFunc
	prevLoad := loadMapPlayerFunc
	inMapCharacterIdsFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model) map[uint32]struct{} {
		return inMap
	}
	loadMapPlayerFunc = func(_ logrus.FieldLogger, _ context.Context, id uint32) (character.Model, error) {
		mc, ok := players[id]
		if !ok {
			return character.Model{}, errors.New("player not found")
		}
		return mc, nil
	}
	t.Cleanup(func() {
		inMapCharacterIdsFunc = prevInMap
		loadMapPlayerFunc = prevLoad
	})
}

func mkPlayerCharAt(id uint32, hp uint16, x, y int16) character.Model {
	return character.NewModelBuilder().SetId(id).SetHp(hp).SetMaxHp(1000).SetX(x).SetY(y).MustBuild()
}

func rectEffect(t *testing.T) effect.Model {
	t.Helper()
	e, err := effect.Extract(effect.RestModel{
		LT: &effect.PointRestModel{X: -400, Y: -350},
		RB: &effect.PointRestModel{X: 400, Y: 250},
	})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return e
}

// TestSelectDeadInRangePartyMembers_KeepsOnlyDead verifies that only dead
// party members (Hp==0) are returned by the Bishop resurrection selector.
// Alive member B (hp=500) must be excluded; dead member A (hp=0) must be included.
// Party: caster=idx0, A=idx1 (dead), B=idx2 (alive).
// Bitmap selects both A and B: bit4 | bit3 = 0b00011000.
func TestSelectDeadInRangePartyMembers_KeepsOnlyDead(t *testing.T) {
	a := mkPartyMember(testMemberA, true, channel.Id(0), _map.Id(40000))
	b := mkPartyMember(testMemberB, true, channel.Id(0), _map.Id(40000))
	installPartySeams(t, threePersonParty(a, b), nil,
		map[uint32]struct{}{testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			testMemberA: mkMemberChar(testMemberA, 0),   // dead
			testMemberB: mkMemberChar(testMemberB, 500), // alive
		},
	)
	bitmap := byte(1<<4 | 1<<3)
	got := SelectDeadInRangePartyMembers(testLogger(), context.Background(), mkField(), testCasterId, 0, 0, rectEffect(t), bitmap)
	if !eqIds(recipientIds(got), []uint32{testMemberA}) {
		t.Fatalf("got %v, want [%d] (dead only)", recipientIds(got), testMemberA)
	}
}

// TestSelectDeadInRangePartyMembers_MissingRectangleReturnsNil verifies that a
// zero-valued effect (no LT/RB rectangle) returns nil — no one to revive.
func TestSelectDeadInRangePartyMembers_MissingRectangleReturnsNil(t *testing.T) {
	got := SelectDeadInRangePartyMembers(testLogger(), context.Background(), mkField(), testCasterId, 0, 0, effect.Model{}, 0x7E)
	if got != nil {
		t.Fatalf("got %v, want nil for missing rectangle", got)
	}
}

// TestSelectDeadInRangeMapPlayers_MissingRectangleReturnsNil verifies that a
// zero-valued effect (no LT/RB rectangle) returns nil — no one to revive.
func TestSelectDeadInRangeMapPlayers_MissingRectangleReturnsNil(t *testing.T) {
	got := SelectDeadInRangeMapPlayers(testLogger(), context.Background(), mkField(), testCasterId, 0, 0, effect.Model{})
	if got != nil {
		t.Fatalf("got %v, want nil for missing rectangle", got)
	}
}

// TestSelectDeadInRangeMapPlayers_AllDeadRegardlessOfParty verifies the GM
// resurrection selector: dead players in range are included regardless of party,
// alive players and the caster are excluded, and out-of-range dead players are excluded.
func TestSelectDeadInRangeMapPlayers_AllDeadRegardlessOfParty(t *testing.T) {
	caster := uint32(1)
	inMap := map[uint32]struct{}{1: {}, 2: {}, 3: {}, 4: {}}
	players := map[uint32]character.Model{
		1: mkPlayerCharAt(1, 800, 0, 0),  // caster (alive) — excluded
		2: mkPlayerCharAt(2, 0, 100, 50), // dead, in range
		3: mkPlayerCharAt(3, 600, 0, 0),  // alive — excluded
		4: mkPlayerCharAt(4, 0, 5000, 0), // dead but out of range — excluded
	}
	installMapSeams(t, inMap, players)
	got := SelectDeadInRangeMapPlayers(testLogger(), context.Background(), mkField(), caster, 0, 0, rectEffect(t))
	if !eqIds(recipientIds(got), []uint32{2}) {
		t.Fatalf("got %v, want [2]", recipientIds(got))
	}
}

// TestSelectDeadInRangeMapPlayers_CapturesDeathCoords verifies that the
// returned recipient carries the character's actual (x, y) coordinates.
func TestSelectDeadInRangeMapPlayers_CapturesDeathCoords(t *testing.T) {
	inMap := map[uint32]struct{}{2: {}}
	players := map[uint32]character.Model{2: mkPlayerCharAt(2, 0, 123, -45)}
	installMapSeams(t, inMap, players)
	got := SelectDeadInRangeMapPlayers(testLogger(), context.Background(), mkField(), 1, 0, 0, rectEffect(t))
	if len(got) != 1 || got[0].X() != 123 || got[0].Y() != -45 {
		t.Fatalf("got %+v, want one recipient at (123,-45)", got)
	}
}

// TestSelectInRangePartyMembers_StillExcludesDead is a regression guard: after
// the wantDead refactor, the living-only selector must still exclude dead members.
func TestSelectInRangePartyMembers_StillExcludesDead(t *testing.T) {
	a := mkPartyMember(testMemberA, true, channel.Id(0), _map.Id(40000))
	b := mkPartyMember(testMemberB, true, channel.Id(0), _map.Id(40000))
	installPartySeams(t, threePersonParty(a, b), nil,
		map[uint32]struct{}{testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			testMemberA: mkMemberChar(testMemberA, 0),   // dead -> excluded
			testMemberB: mkMemberChar(testMemberB, 500), // alive -> included
		},
	)
	bitmap := byte(1<<4 | 1<<3)
	got := SelectInRangePartyMembers(testLogger(), context.Background(), mkField(), testCasterId, 0, 0, rectEffect(t), bitmap)
	if !eqIds(recipientIds(got), []uint32{testMemberB}) {
		t.Fatalf("got %v, want [%d] (alive only)", recipientIds(got), testMemberB)
	}
}
