package hpsync

import (
	"context"
	"io"
	"sort"
	"sync"
	"testing"

	"atlas-channel/character"
	"atlas-channel/party"
	"atlas-channel/socket/writer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func mkField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
}

func mkMember(id uint32, online bool, ch channel.Id, mapId _map.Id) party.MemberModel {
	m, _ := party.ExtractMember(party.MemberRestModel{
		Id:        id,
		Name:      "m",
		WorldId:   world.Id(0),
		ChannelId: ch,
		MapId:     mapId,
		Instance:  uuid.Nil,
		Online:    online,
	})
	return m
}

// ann captures one announceMemberHPFunc call.
type ann struct {
	to      uint32
	subject uint32
	hp      uint16
	maxHp   uint16
}

type capture struct {
	mu   sync.Mutex
	anns []ann
}

func (c *capture) record(to, subject uint32, hp, maxHp uint16) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.anns = append(c.anns, ann{to: to, subject: subject, hp: hp, maxHp: maxHp})
}

func (c *capture) sorted() []ann {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := append([]ann(nil), c.anns...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].to != out[j].to {
			return out[i].to < out[j].to
		}
		return out[i].subject < out[j].subject
	})
	return out
}

// installSeams replaces the three external-dependency seams. partyChar is what
// loadPartyCharacterFunc returns; members maps member id -> character.Model (or
// is consulted for an error via memberErr).
func installSeams(t *testing.T, c *capture, partyChar character.Model, partyErr error, members map[uint32]character.Model, memberErr map[uint32]error) {
	t.Helper()
	prevParty := loadPartyCharacterFunc
	prevMember := loadCharacterFunc
	prevAnnounce := announceMemberHPFunc

	loadPartyCharacterFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (character.Model, error) {
		return partyChar, partyErr
	}
	loadCharacterFunc = func(_ logrus.FieldLogger, _ context.Context, id uint32) (character.Model, error) {
		if memberErr != nil {
			if e, ok := memberErr[id]; ok {
				return character.Model{}, e
			}
		}
		return members[id], nil
	}
	announceMemberHPFunc = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ channel.Model, to uint32, subject uint32, hp uint16, maxHp uint16) error {
		c.record(to, subject, hp, maxHp)
		return nil
	}

	t.Cleanup(func() {
		loadPartyCharacterFunc = prevParty
		loadCharacterFunc = prevMember
		announceMemberHPFunc = prevAnnounce
	})
}

func mkChar(id uint32, hp, maxHp uint16) character.Model {
	return character.NewModelBuilder().SetId(id).SetHp(hp).SetMaxHp(maxHp).MustBuild()
}

func mkPartyChar(id uint32, hp, maxHp uint16, members []party.MemberModel) character.Model {
	p := party.NewBuilder().SetId(1).SetLeaderId(id).SetMembers(members).Build()
	return character.NewModelBuilder().SetId(id).SetHp(hp).SetMaxHp(maxHp).SetParty(p).MustBuild()
}

const (
	caster  = uint32(100)
	memberA = uint32(101)
	memberB = uint32(102)
)

func TestSync_BidirectionalForInMapMembers(t *testing.T) {
	c := &capture{}
	pc := mkPartyChar(caster, 800, 1000, []party.MemberModel{
		mkMember(caster, true, channel.Id(0), _map.Id(40000)),
		mkMember(memberA, true, channel.Id(0), _map.Id(40000)),
		mkMember(memberB, true, channel.Id(0), _map.Id(40000)),
	})
	installSeams(t, c, pc, nil,
		map[uint32]character.Model{
			memberA: mkChar(memberA, 100, 200),
			memberB: mkChar(memberB, 300, 400),
		}, nil)

	if err := Sync(testLogger(), context.Background(), nil, mkField(), caster); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	got := c.sorted()
	want := []ann{
		{to: caster, subject: memberA, hp: 100, maxHp: 200},  // A's HP -> me
		{to: caster, subject: memberB, hp: 300, maxHp: 400},  // B's HP -> me
		{to: memberA, subject: caster, hp: 800, maxHp: 1000}, // my HP -> A
		{to: memberB, subject: caster, hp: 800, maxHp: 1000}, // my HP -> B
	}
	if len(got) != len(want) {
		t.Fatalf("got %d announces, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("announce[%d] = %+v, want %+v (all: %+v)", i, got[i], want[i], got)
		}
	}
}

func TestSync_NotInPartyIsNoOp(t *testing.T) {
	c := &capture{}
	// No SetParty -> party.Id() == 0 -> InParty() false.
	pc := character.NewModelBuilder().SetId(caster).SetHp(800).SetMaxHp(1000).MustBuild()
	installSeams(t, c, pc, nil, map[uint32]character.Model{}, nil)

	if err := Sync(testLogger(), context.Background(), nil, mkField(), caster); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if got := c.sorted(); len(got) != 0 {
		t.Fatalf("expected no announces, got %+v", got)
	}
}

func TestSync_ExcludesOutOfMapMember(t *testing.T) {
	c := &capture{}
	pc := mkPartyChar(caster, 800, 1000, []party.MemberModel{
		mkMember(caster, true, channel.Id(0), _map.Id(40000)),
		mkMember(memberA, true, channel.Id(0), _map.Id(40000)),
		mkMember(memberB, true, channel.Id(0), _map.Id(99999)), // different map
	})
	installSeams(t, c, pc, nil,
		map[uint32]character.Model{
			memberA: mkChar(memberA, 100, 200),
			memberB: mkChar(memberB, 300, 400),
		}, nil)

	if err := Sync(testLogger(), context.Background(), nil, mkField(), caster); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	got := c.sorted()
	want := []ann{
		{to: caster, subject: memberA, hp: 100, maxHp: 200},
		{to: memberA, subject: caster, hp: 800, maxHp: 1000},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d announces, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("announce[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestSync_StaleMemberSkipsInboundOnly(t *testing.T) {
	c := &capture{}
	pc := mkPartyChar(caster, 800, 1000, []party.MemberModel{
		mkMember(caster, true, channel.Id(0), _map.Id(40000)),
		mkMember(memberB, true, channel.Id(0), _map.Id(40000)),
	})
	installSeams(t, c, pc, nil,
		map[uint32]character.Model{},
		map[uint32]error{memberB: requests.ErrNotFound})

	if err := Sync(testLogger(), context.Background(), nil, mkField(), caster); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	// Outbound (my HP -> B) still fires; inbound (B's HP -> me) is skipped
	// because B's character record is gone.
	got := c.sorted()
	want := []ann{{to: memberB, subject: caster, hp: 800, maxHp: 1000}}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestSync_PartyLoadErrorPropagates(t *testing.T) {
	c := &capture{}
	installSeams(t, c, character.Model{}, requests.ErrNotFound, map[uint32]character.Model{}, nil)

	if err := Sync(testLogger(), context.Background(), nil, mkField(), caster); err == nil {
		t.Fatal("expected error from party-character load, got nil")
	}
	if got := c.sorted(); len(got) != 0 {
		t.Fatalf("expected no announces on load error, got %+v", got)
	}
}
