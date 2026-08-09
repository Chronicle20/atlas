package trade

import (
	"testing"

	"github.com/google/uuid"
)

// TestBuilderSeatsOwnerAtPositionZero pins that a freshly built room is solo,
// OPEN_SOLO, and that the handle defaults to the owner's character id
// (design §2.3).
func TestBuilderSeatsOwnerAtPositionZero(t *testing.T) {
	room := NewBuilder(3, 100, "Owner", testField(t)).Build()

	if got := room.State(); got != StateOpenSolo {
		t.Errorf("State() = %v, want %v", got, StateOpenSolo)
	}
	if got := room.Handle(); got != 100 {
		t.Errorf("Handle() = %d, want 100 (owner id)", got)
	}
	if got := room.OwnerId(); got != 100 {
		t.Errorf("OwnerId() = %d, want 100", got)
	}
	if got := room.VisitorId(); got != 0 {
		t.Errorf("VisitorId() = %d, want 0 on a solo room", got)
	}
	if got := len(room.Participants()); got != 1 {
		t.Errorf("len(Participants()) = %d, want 1", got)
	}
	if room.Id() == uuid.Nil {
		t.Error("Id() is the nil uuid; the builder must mint one")
	}
	if room.RoomType() != 3 {
		t.Errorf("RoomType() = %d, want 3", room.RoomType())
	}
	if !room.Field().Equals(testField(t)) {
		t.Errorf("Field() = %v, want the field passed to NewBuilder", room.Field())
	}
	if room.CreatedAt().IsZero() {
		t.Error("CreatedAt() is zero; the builder must stamp it")
	}
}

// TestBuilderSetVisitorSeatsPositionOne pins that SetVisitor adds the invited
// character at position 1 and leaves the owner at position 0.
func TestBuilderSetVisitorSeatsPositionOne(t *testing.T) {
	room := NewBuilder(6, 100, "Owner", testField(t)).SetVisitor(200, "Guest").Build()

	owner, ok := room.ParticipantFor(100)
	if !ok {
		t.Fatal("ParticipantFor(100): not found")
	}
	if owner.Position() != 0 || owner.Name() != "Owner" {
		t.Errorf("owner = position %d name %q, want position 0 name \"Owner\"", owner.Position(), owner.Name())
	}

	visitor, ok := room.ParticipantFor(200)
	if !ok {
		t.Fatal("ParticipantFor(200): not found")
	}
	if visitor.Position() != 1 || visitor.Name() != "Guest" {
		t.Errorf("visitor = position %d name %q, want position 1 name \"Guest\"", visitor.Position(), visitor.Name())
	}
	if got := room.VisitorId(); got != 200 {
		t.Errorf("VisitorId() = %d, want 200", got)
	}
}

// TestParticipantForUnknownCharacter pins the miss path — a packet arriving
// from a character who is not in the room must not resolve a participant.
func TestParticipantForUnknownCharacter(t *testing.T) {
	room := NewBuilder(3, 100, "Owner", testField(t)).SetVisitor(200, "Guest").Build()

	if _, ok := room.ParticipantFor(999); ok {
		t.Error("ParticipantFor(999) resolved a character that is not in the room")
	}
}

// TestRoomFrozenStates pins the freeze rule (FR-3.6, design §3.2): staging is
// open only in OPEN with neither side confirmed. Every other state, and OPEN
// after the FIRST confirm, is frozen.
func TestRoomFrozenStates(t *testing.T) {
	base := func(s State) Room {
		return NewBuilder(3, 100, "Owner", testField(t)).SetVisitor(200, "Guest").SetState(s).Build()
	}

	for _, s := range []State{StateOpenSolo, StatePendingInvite, StateAwaitingAttestation, StateSettling} {
		if !base(s).Frozen() {
			t.Errorf("Frozen() = false in state %v, want true", s)
		}
	}

	open := base(StateOpen)
	if open.Frozen() {
		t.Error("Frozen() = true in OPEN with nobody confirmed, want false")
	}

	ownerConfirmed := open.WithParticipant(0, func(p Participant) Participant { return p.WithConfirmed(true) })
	if !ownerConfirmed.Frozen() {
		t.Error("Frozen() = false after the owner confirmed, want true")
	}

	visitorConfirmed := open.WithParticipant(1, func(p Participant) Participant { return p.WithConfirmed(true) })
	if !visitorConfirmed.Frozen() {
		t.Error("Frozen() = false after the visitor confirmed, want true")
	}
}

// TestWithStateLeavesTheOriginalUntouched pins that WithState is a copy, not a
// mutation — the registry relies on the pre-image staying valid when a
// compare-and-set loses.
func TestWithStateLeavesTheOriginalUntouched(t *testing.T) {
	room := NewBuilder(3, 100, "Owner", testField(t)).SetState(StateOpen).Build()

	settling := room.WithState(StateSettling)

	if room.State() != StateOpen {
		t.Errorf("original State() = %v, want %v", room.State(), StateOpen)
	}
	if settling.State() != StateSettling {
		t.Errorf("copy State() = %v, want %v", settling.State(), StateSettling)
	}
	if settling.Id() != room.Id() {
		t.Error("WithState changed the room id")
	}
}

// TestWithParticipantLeavesTheOriginalUntouched pins that WithParticipant
// copies the participant slice rather than writing into the shared backing
// array of the room it was called on.
func TestWithParticipantLeavesTheOriginalUntouched(t *testing.T) {
	room := NewBuilder(3, 100, "Owner", testField(t)).SetVisitor(200, "Guest").SetState(StateOpen).Build()

	updated := room.WithParticipant(1, func(p Participant) Participant {
		return p.WithMesoStaged(5000).WithConfirmed(true)
	})

	before, _ := room.ParticipantFor(200)
	if before.MesoStaged() != 0 || before.Confirmed() {
		t.Errorf("original visitor mutated: meso %d confirmed %v", before.MesoStaged(), before.Confirmed())
	}

	after, _ := updated.ParticipantFor(200)
	if after.MesoStaged() != 5000 || !after.Confirmed() {
		t.Errorf("copy visitor = meso %d confirmed %v, want 5000 / true", after.MesoStaged(), after.Confirmed())
	}

	ownerAfter, _ := updated.ParticipantFor(100)
	if ownerAfter.MesoStaged() != 0 || ownerAfter.Confirmed() {
		t.Error("WithParticipant(1, ...) touched position 0")
	}
}

// TestWithParticipantOnAbsentPositionIsANoOp pins that transforming a position
// nobody occupies (position 1 of a solo room) leaves the room unchanged rather
// than fabricating a participant.
func TestWithParticipantOnAbsentPositionIsANoOp(t *testing.T) {
	room := NewBuilder(3, 100, "Owner", testField(t)).SetState(StateOpen).Build()

	updated := room.WithParticipant(1, func(p Participant) Participant { return p.WithConfirmed(true) })

	if got := len(updated.Participants()); got != 1 {
		t.Fatalf("len(Participants()) = %d, want 1", got)
	}
	if updated.Frozen() {
		t.Error("Frozen() = true; a no-op transform must not confirm anyone")
	}
}

// TestParticipantWithItemAndHasTradeSlot pins staged-item accumulation and the
// duplicate-slot probe (FR-3.3), including that WithItem does not write into
// the receiver's item slice.
func TestParticipantWithItemAndHasTradeSlot(t *testing.T) {
	room := NewBuilder(3, 100, "Owner", testField(t)).SetState(StateOpen).Build()
	owner, _ := room.ParticipantFor(100)

	if owner.HasTradeSlot(1) {
		t.Error("HasTradeSlot(1) = true on a participant with no staged items")
	}

	one := owner.WithItem(NewStagedItem(1, 4001, 2000000, 3, 2, 5))
	two := one.WithItem(NewStagedItem(4, 4002, 1302000, 1, 1, 9))

	if len(owner.Items()) != 0 {
		t.Errorf("original participant gained %d items", len(owner.Items()))
	}
	if len(one.Items()) != 1 {
		t.Errorf("len(one.Items()) = %d, want 1", len(one.Items()))
	}
	if len(two.Items()) != 2 {
		t.Errorf("len(two.Items()) = %d, want 2", len(two.Items()))
	}

	if !two.HasTradeSlot(1) || !two.HasTradeSlot(4) {
		t.Error("HasTradeSlot missed an occupied slot")
	}
	if two.HasTradeSlot(2) {
		t.Error("HasTradeSlot(2) = true for an unoccupied slot")
	}

	got := two.Items()[0]
	if got.TradeSlot() != 1 || got.AssetId() != 4001 || got.TemplateId() != 2000000 ||
		got.Quantity() != 3 || got.InventoryType() != 2 || got.SourceSlot() != 5 {
		t.Errorf("staged item round-trip = %+v, want the values passed to NewStagedItem", got)
	}
}

// TestItemsIsNotWritableThroughTheGetter pins that a caller cannot reach into
// a participant's staged items through the returned slice.
func TestItemsIsNotWritableThroughTheGetter(t *testing.T) {
	room := NewBuilder(3, 100, "Owner", testField(t)).SetState(StateOpen).Build()
	owner, _ := room.ParticipantFor(100)
	owner = owner.WithItem(NewStagedItem(1, 4001, 2000000, 3, 2, 5))

	escaped := owner.Items()
	escaped[0] = NewStagedItem(9, 0, 0, 0, 0, 0)

	if owner.Items()[0].TradeSlot() != 1 {
		t.Error("writing through Items() mutated the participant")
	}
}

// TestParticipantsIsNotWritableThroughTheGetter pins the same for the room's
// participant list.
func TestParticipantsIsNotWritableThroughTheGetter(t *testing.T) {
	room := NewBuilder(3, 100, "Owner", testField(t)).SetVisitor(200, "Guest").Build()

	escaped := room.Participants()
	escaped[0] = Participant{}

	if room.OwnerId() != 100 {
		t.Error("writing through Participants() mutated the room")
	}
}
