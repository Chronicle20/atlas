package door

import (
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// helper: decode (type, owner, partyId, forCharacterId) for an emitted event.
func decodeEvt(b []byte) (typ string, owner, party, forCh uint32) {
	var env struct {
		Type             string `json:"type"`
		OwnerCharacterId uint32 `json:"ownerCharacterId"`
		PartyId          uint32 `json:"partyId"`
		ForCharacterId   uint32 `json:"forCharacterId"`
	}
	_ = json.Unmarshal(b, &env)
	return env.Type, env.OwnerCharacterId, env.PartyId, env.ForCharacterId
}

func twoPartyPortals() []TownPortal {
	return []TownPortal{{X: 10, Y: 20}, {X: -85, Y: 531}}
}

func TestReconcileExpelDropsLeaverToSoloAndCrossHides(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()

	// Leader Chronicle (1) slot 0, Bishop (5) slot 1 — both in party 1000000008.
	chron := NewBuilder().SetAreaDoorId(1).SetTownDoorId(2).SetOwnerCharacterId(1).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).
		SetTownPortalId(0x80).SetTownX(10).SetTownY(20).Build()
	bishop := NewBuilder().SetAreaDoorId(3).SetTownDoorId(4).SetOwnerCharacterId(5).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(1).
		SetTownPortalId(0x81).SetTownX(-85).SetTownY(531).Build()
	for _, m := range []Model{chron, bishop} {
		if err := GetRegistry().Put(ctx, ten, m); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// Chronicle expels Bishop: members=[1], leavers=[5].
	if err := ReconcileParty(p, 1000000008, []character.Id{1}, nil, []character.Id{5},
		func(_ _map.Id) []TownPortal { return twoPartyPortals() }); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	// Expect: Bishop's door REMOVED from Chronicle (forCh=1), Bishop solo CREATED (forCh=0),
	// Chronicle's door REMOVED from Bishop (forCh=5). Order: drop-to-solo first, then hide.
	gotRemovedFromChron, gotBishopSolo, gotRemovedFromBishop := false, false, false
	for _, v := range em.values {
		typ, owner, party, forCh := decodeEvt(v)
		if typ == EventDoorStatusRemoved && owner == 5 && party == 1000000008 && forCh == 1 {
			gotRemovedFromChron = true
		}
		if typ == EventDoorStatusCreated && owner == 5 && party == 0 && forCh == 0 {
			gotBishopSolo = true
		}
		if typ == EventDoorStatusRemoved && owner == 1 && party == 1000000008 && forCh == 5 {
			gotRemovedFromBishop = true
		}
	}
	if !gotRemovedFromChron || !gotBishopSolo || !gotRemovedFromBishop {
		t.Fatalf("expel deltas missing: rmFromChron=%v bishopSolo=%v rmFromBishop=%v events=%v",
			gotRemovedFromChron, gotBishopSolo, gotRemovedFromBishop, em.types)
	}
	// Bishop's door is now solo at slot 0.
	got, _ := GetRegistry().Get(ctx, ten, 3)
	if got.PartyId() != 0 || got.Slot() != 0 {
		t.Fatalf("bishop door not solo: party=%d slot=%d", got.PartyId(), got.Slot())
	}
}

func TestReconcileDisbandDropsAllAndCrossRemoves(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()
	chron := NewBuilder().SetAreaDoorId(1).SetTownDoorId(2).SetOwnerCharacterId(1).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build()
	bishop := NewBuilder().SetAreaDoorId(3).SetTownDoorId(4).SetOwnerCharacterId(5).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(1).Build()
	for _, m := range []Model{chron, bishop} {
		_ = GetRegistry().Put(ctx, ten, m)
	}

	// Leader leaves -> disband: members empty, leavers=[1,5] (the atlas-parties fix supplies both).
	_ = ReconcileParty(p, 1000000008, nil, nil, []character.Id{1, 5},
		func(_ _map.Id) []TownPortal { return twoPartyPortals() })

	// Each door removed from the OTHER former member; each re-keyed solo.
	rmChronFromBishop, rmBishopFromChron := false, false
	for _, v := range em.values {
		typ, owner, party, forCh := decodeEvt(v)
		if typ == EventDoorStatusRemoved && owner == 1 && party == 1000000008 && forCh == 5 {
			rmChronFromBishop = true
		}
		if typ == EventDoorStatusRemoved && owner == 5 && party == 1000000008 && forCh == 1 {
			rmBishopFromChron = true
		}
	}
	if !rmChronFromBishop || !rmBishopFromChron {
		t.Fatalf("disband cross-removal missing: chron->bishop=%v bishop->chron=%v events=%v",
			rmChronFromBishop, rmBishopFromChron, em.types)
	}
	for _, id := range []uint32{1, 3} {
		got, _ := GetRegistry().Get(ctx, ten, id)
		if got.PartyId() != 0 || got.Slot() != 0 {
			t.Fatalf("door %d not solo after disband: party=%d slot=%d", id, got.PartyId(), got.Slot())
		}
	}
}

func TestReconcileReinviteAdoptsWithoutOwnerAreaResend(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()
	// Chronicle (1) in party; Bishop (5) currently SOLO (post-expel).
	chron := NewBuilder().SetAreaDoorId(1).SetTownDoorId(2).SetOwnerCharacterId(1).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build()
	bishopSolo := NewBuilder().SetAreaDoorId(3).SetTownDoorId(4).SetOwnerCharacterId(5).
		SetPartyId(0).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build()
	for _, m := range []Model{chron, bishopSolo} {
		_ = GetRegistry().Put(ctx, ten, m)
	}

	// Bishop rejoins: members=[1,5], joiners=[5].
	_ = ReconcileParty(p, 1000000008, []character.Id{1, 5}, []character.Id{5}, nil,
		func(_ _map.Id) []TownPortal { return twoPartyPortals() })

	// FLICKER GUARD: Bishop (owner 5) must NOT receive a CREATED for his OWN door (owner 5).
	for _, v := range em.values {
		typ, owner, _, forCh := decodeEvt(v)
		if typ == EventDoorStatusCreated && owner == 5 && forCh == 5 {
			t.Fatalf("owner 5 got a CREATED for his own door (area re-send / platform-below flicker): %v", em.types)
		}
	}
	// Chronicle gains Bishop's door (CREATED owner 5 -> forCh 1); Bishop gains Chronicle's door (CREATED owner 1 -> forCh 5).
	chronGainsBishop, bishopGainsChron := false, false
	for _, v := range em.values {
		typ, owner, _, forCh := decodeEvt(v)
		if typ == EventDoorStatusCreated && owner == 5 && forCh == 1 {
			chronGainsBishop = true
		}
		if typ == EventDoorStatusCreated && owner == 1 && forCh == 5 {
			bishopGainsChron = true
		}
	}
	if !chronGainsBishop || !bishopGainsChron {
		t.Fatalf("reinvite visibility missing: chronGainsBishop=%v bishopGainsChron=%v events=%v",
			chronGainsBishop, bishopGainsChron, em.types)
	}
	// Bishop's door is back in the party at slot 1.
	got, _ := GetRegistry().Get(ctx, ten, 3)
	if got.PartyId() != 1000000008 || got.Slot() != 1 {
		t.Fatalf("bishop door not re-adopted at slot 1: party=%d slot=%d", got.PartyId(), got.Slot())
	}
}

func TestReconcileIsIdempotent(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()
	chron := NewBuilder().SetAreaDoorId(1).SetTownDoorId(2).SetOwnerCharacterId(1).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build()
	bishop := NewBuilder().SetAreaDoorId(3).SetTownDoorId(4).SetOwnerCharacterId(5).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(1).Build()
	for _, m := range []Model{chron, bishop} {
		_ = GetRegistry().Put(ctx, ten, m)
	}
	// Steady-state reconcile (no joiners/leavers, slots already correct) emits nothing.
	_ = ReconcileParty(p, 1000000008, []character.Id{1, 5}, nil, nil,
		func(_ _map.Id) []TownPortal { return twoPartyPortals() })
	if len(em.types) != 0 {
		t.Fatalf("steady-state reconcile must emit nothing, got %v", em.types)
	}
}

func TestReconcileHealsOrphanTaggedToDeadParty(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()
	// Owner 1 is a current member of party 1000000009, but their door is still
	// tagged to a DEAD party 1000000008 (orphan).
	orphan := NewBuilder().SetAreaDoorId(1).SetTownDoorId(2).SetOwnerCharacterId(1).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build()
	_ = GetRegistry().Put(ctx, ten, orphan)

	_ = ReconcileParty(p, 1000000009, []character.Id{1}, nil, nil,
		func(_ _map.Id) []TownPortal { return twoPartyPortals() })

	got, _ := GetRegistry().Get(ctx, ten, 1)
	if got.PartyId() != 1000000009 {
		t.Fatalf("orphan not healed into current party: party=%d", got.PartyId())
	}
}

// TestReconcileAdoptDoesNotClearAnotherMembersSlot asserts that the adopt path
// never emits a SLOT_CHANGED whose OldSlot differs from NewSlot.  In the
// reinvite scenario below, Chronicle is leader at slot 0; Bishop is solo (also
// at slot 0 while solo) and gets adopted at slot 1.  The pre-fix code passed
// oldSlot=0 (Bishop's solo slot) to slotChangedEventProvider, which would emit
// OldSlot=0, NewSlot=1 — telling the channel to clear slot 0 (the leader's
// slot) before setting slot 1.  After the fix, OldSlot must equal NewSlot so
// only Bishop's own slot is touched.
func TestReconcileAdoptDoesNotClearAnotherMembersSlot(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()

	// Chronicle (owner=1) in party at slot 0; Bishop (owner=5) currently solo (slot 0).
	chron := NewBuilder().SetAreaDoorId(1).SetTownDoorId(2).SetOwnerCharacterId(1).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build()
	bishopSolo := NewBuilder().SetAreaDoorId(3).SetTownDoorId(4).SetOwnerCharacterId(5).
		SetPartyId(0).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build()
	for _, m := range []Model{chron, bishopSolo} {
		_ = GetRegistry().Put(ctx, ten, m)
	}

	// Bishop rejoins: members=[1,5], joiners=[5].
	_ = ReconcileParty(p, 1000000008, []character.Id{1, 5}, []character.Id{5}, nil,
		func(_ _map.Id) []TownPortal { return twoPartyPortals() })

	// Any emitted SLOT_CHANGED must have OldSlot == NewSlot (no foreign-slot clear).
	for i, ty := range em.types {
		if ty != EventDoorStatusSlotChanged {
			continue
		}
		var env struct {
			Body struct {
				OldSlot byte `json:"oldSlot"`
				NewSlot byte `json:"newSlot"`
			} `json:"body"`
		}
		_ = json.Unmarshal(em.values[i], &env)
		if env.Body.OldSlot != env.Body.NewSlot {
			t.Fatalf("adopt emitted a foreign-slot clear: OldSlot=%d NewSlot=%d (would wipe another member's slot)", env.Body.OldSlot, env.Body.NewSlot)
		}
	}
}

func TestReconcileNeverEmitsSlotAbove5(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()
	members := []character.Id{1, 2, 3, 4, 5, 6} // full 6-cap party
	for i, owner := range members {
		_ = GetRegistry().Put(ctx, ten, NewBuilder().
			SetAreaDoorId(uint32(10+i)).SetTownDoorId(uint32(100+i)).SetOwnerCharacterId(owner).
			SetPartyId(0).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build())
	}
	_ = ReconcileParty(p, 1000000010, members, members, nil,
		func(_ _map.Id) []TownPortal { return []TownPortal{{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}, {6, 6}} })

	for _, v := range em.values {
		var env struct {
			Body struct {
				Slot    byte `json:"slot"`
				NewSlot byte `json:"newSlot"`
			} `json:"body"`
		}
		_ = json.Unmarshal(v, &env)
		if env.Body.Slot > 5 || env.Body.NewSlot > 5 {
			t.Fatalf("emitted slot > 5 (client-kill): slot=%d newSlot=%d", env.Body.Slot, env.Body.NewSlot)
		}
	}
}
