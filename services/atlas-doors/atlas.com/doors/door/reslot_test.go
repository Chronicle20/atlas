package door

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// TestReslotParty_TwoMembersGetDistinctSlots pins the warp-destination fix: two
// party members' doors (both cast at slot 0) must reslot to distinct town
// portals — leader at slot 0, member at slot 1 — so they warp to different town
// positions instead of both landing on portal index 0.
func TestReslotParty_TwoMembersGetDistinctSlots(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()

	for i, owner := range []character.Id{1, 5} {
		_ = GetRegistry().Put(ctx, ten, NewBuilder().
			SetAreaDoorId(uint32(10+i)).SetTownDoorId(uint32(100+i)).SetOwnerCharacterId(owner).
			SetPartyId(1000).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).
			SetTownPortalId(0x80).SetTownX(10).SetTownY(20).Build())
	}

	// Town door-type portals: slot 0 at (10,20), slot 1 at (99,199).
	portals := []TownPortal{{X: 10, Y: 20}, {X: 99, Y: 199}}
	if err := ReslotParty(p, 1000, []character.Id{1, 5}, nil, func(_ _map.Id) []TownPortal { return portals }); err != nil {
		t.Fatalf("ReslotParty: %v", err)
	}

	// Member 5 (index 1) -> slot 1 at the slot-1 town portal.
	d5, _ := GetRegistry().Get(ctx, ten, 11)
	if d5.Slot() != 1 || d5.TownX() != 99 || d5.TownY() != 199 || d5.TownPortalId() != 0x81 {
		t.Fatalf("member 5 not reslotted to slot 1: slot=%d townX=%d townY=%d portal=%#x",
			d5.Slot(), d5.TownX(), d5.TownY(), d5.TownPortalId())
	}
	// Leader 1 (index 0) stays at slot 0.
	d1, _ := GetRegistry().Get(ctx, ten, 10)
	if d1.Slot() != 0 {
		t.Fatalf("leader 1 should stay slot 0, got %d", d1.Slot())
	}
}

// TestReslotParty_LeaverDropsToSolo: a leaver's door reslots back to solo (slot 0).
func TestReslotParty_LeaverDropsToSolo(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()

	_ = GetRegistry().Put(ctx, ten, NewBuilder().
		SetAreaDoorId(20).SetTownDoorId(200).SetOwnerCharacterId(5).
		SetPartyId(1000).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(1).
		SetTownPortalId(0x81).SetTownX(99).SetTownY(199).Build())

	portals := []TownPortal{{X: 10, Y: 20}, {X: 99, Y: 199}}
	_ = ReslotParty(p, 1000, nil, []character.Id{5}, func(_ _map.Id) []TownPortal { return portals })

	d, _ := GetRegistry().Get(ctx, ten, 20)
	if d.Slot() != 0 || d.TownX() != 10 || d.TownY() != 20 {
		t.Fatalf("leaver door not reslotted to solo slot 0: slot=%d townX=%d townY=%d", d.Slot(), d.TownX(), d.TownY())
	}
}
