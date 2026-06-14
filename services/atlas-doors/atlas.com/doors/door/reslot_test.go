package door

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

// TestReslotPartyRecomputesChangedSlots covers FR-4.3 / FR-6.4.
//
// Scenario: party [A=1, B=2, C=3] (slots 0/1/2).
//   - A has a door at slot 0.
//   - C has a door at slot 2.
//
// After A leaves, newMembers=[B,C], formerMembers=[A].
//   - C slot: ComputeSlot(partyId, [B,C], C) = 1  → changed from 2 → SLOT_CHANGED
//   - A slot: solo → 0  → unchanged from 0? No — A still has a door stored with
//     partyId=42 and slot=0; after leaving the party it must drop to solo-scope (slot 0).
//     But slot 0 == old slot 0 → the Reslot no-op fires, no SLOT_CHANGED for A.
//
// Key assertions:
//  1. Exactly ONE SLOT_CHANGED emitted — for C.
//  2. No SLOT_CHANGED for A (old slot 0, new solo slot 0 → Reslot no-op).
//  3. Registry updated: C's door now has slot=1 and townPortalId=0x81.
func TestReslotPartyRecomputesChangedSlots(t *testing.T) {
	const (
		partyId = uint32(42)
		charA   = uint32(1)
		charB   = uint32(2)
		charC   = uint32(3)
	)

	em := &fakeEmit{}
	ten, ctx := newTestTenant()
	GetRegistry().Clear(ctx)

	srcField := field.NewBuilder(1, 2, 100000000).Build()
	townMapId := _map.Id(104000000)

	// Seed A's door: slot 0, partyId=42.
	doorA := NewBuilder().
		SetAreaDoorId(1_001_001).SetTownDoorId(1_001_002).
		SetOwnerCharacterId(charA).SetPartyId(partyId).
		SetField(srcField).SetTownMapId(townMapId).
		SetSlot(0).SetTownPortalId(0x80).
		Build()
	if err := GetRegistry().Put(ctx, ten, doorA); err != nil {
		t.Fatalf("seed A: %v", err)
	}

	// Seed C's door: slot 2, partyId=42.
	doorC := NewBuilder().
		SetAreaDoorId(1_003_001).SetTownDoorId(1_003_002).
		SetOwnerCharacterId(charC).SetPartyId(partyId).
		SetField(srcField).SetTownMapId(townMapId).
		SetSlot(2).SetTownPortalId(0x82).
		Build()
	if err := GetRegistry().Put(ctx, ten, doorC); err != nil {
		t.Fatalf("seed C: %v", err)
	}

	// Town portals for townMapId: three door-type portals at known positions.
	townPortals := []TownPortal{
		{X: 100, Y: 10}, // slot 0 → wireId 0x80
		{X: 200, Y: 10}, // slot 1 → wireId 0x81
		{X: 300, Y: 10}, // slot 2 → wireId 0x82
	}
	townPortalsByMap := func(id _map.Id) []TownPortal {
		if id == townMapId {
			return townPortals
		}
		return nil
	}

	// Build a processor with fake emit and no alloc (reslot doesn't allocate).
	p := &ProcessorImpl{
		l:     logrusLogger(),
		ctx:   ctx,
		t:     ten,
		emit:  em.emit,
		res:   fakeResolver{},
		alloc: &counterAllocator{next: 9_000_001},
	}

	// newMembers=[B,C], formerMembers=[A].
	newMembers := []uint32{charB, charC}
	formerMembers := []uint32{charA}

	if err := ReslotParty(p, partyId, newMembers, formerMembers, townPortalsByMap); err != nil {
		t.Fatalf("ReslotParty: %v", err)
	}

	// Exactly one SLOT_CHANGED (for C; A's slot 0→0 is a no-op).
	var slotChangedCount int
	for _, ty := range em.types {
		if ty == EventDoorStatusSlotChanged {
			slotChangedCount++
		}
	}
	if slotChangedCount != 1 {
		t.Fatalf("expected exactly 1 SLOT_CHANGED, got %d (all events: %v)", slotChangedCount, em.types)
	}

	// C's door must now have slot=1, townPortalId=0x81, townX=200.
	gotC, err := GetRegistry().Get(ctx, ten, 1_003_001)
	if err != nil {
		t.Fatalf("Get C after reslot: %v", err)
	}
	if gotC.Slot() != 1 {
		t.Fatalf("C slot: want 1, got %d", gotC.Slot())
	}
	if gotC.TownPortalId() != 0x81 {
		t.Fatalf("C townPortalId: want 0x81, got 0x%x", gotC.TownPortalId())
	}
	if gotC.TownX() != 200 {
		t.Fatalf("C townX: want 200, got %d", gotC.TownX())
	}

	// A's door slot still 0 (no-op from Reslot); no SLOT_CHANGED for A.
	gotA, err := GetRegistry().Get(ctx, ten, 1_001_001)
	if err != nil {
		t.Fatalf("Get A after reslot: %v", err)
	}
	if gotA.Slot() != 0 {
		t.Fatalf("A slot: want 0, got %d", gotA.Slot())
	}
}

// logrusLogger returns a logrus logger for tests (no output noise).
func logrusLogger() *logrus.Logger {
	return logrus.New()
}
