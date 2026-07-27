package game

import (
	"atlas-mini-games/game/omok"
	"atlas-mini-games/record"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
}

func TestBuilder_Defaults(t *testing.T) {
	r := NewBuilder(1, 1001, testField()).Build()

	if r.FirstMover() != 1 {
		t.Errorf("FirstMover() = %d, want 1", r.FirstMover())
	}
	if r.FirstSlot() != -1 {
		t.Errorf("FirstSlot() = %d, want -1", r.FirstSlot())
	}
	if r.RoomType() != 1 {
		t.Errorf("RoomType() = %d, want 1", r.RoomType())
	}
	if r.OwnerId() != 1001 {
		t.Errorf("OwnerId() = %d, want 1001", r.OwnerId())
	}
	if r.Id() != r.OwnerId() {
		t.Errorf("Id() = %d, want == OwnerId() %d", r.Id(), r.OwnerId())
	}
}

func TestBuilder_SetEveryField_RoundTrips(t *testing.T) {
	f := testField()
	until := time.Now().Add(5 * time.Minute)
	board := [omok.Cells]byte{}
	board[10] = 2
	moves := []Move{{X: 1, Y: 2, Stone: 1}}
	deck := []uint32{0, 0, 1, 1}

	r := NewBuilder(2, 2002, f).
		SetTitle("Room Title").
		SetPrivate(true).
		SetPassword("secret").
		SetPieceType(3).
		SetVisitorId(3003).
		SetVisitorReady(true).
		SetInProgress(true).
		SetDeniedTie(0, true).
		SetDeniedTie(1, true).
		SetExitAfter(0, true).
		SetExitAfter(1, true).
		SetFirstMover(0).
		SetCurrentTurn(1).
		SetBoard(board).
		SetMoves(moves).
		SetDeck(deck).
		SetFirstSlot(1).
		SetOwnerPairs(4).
		SetVisitorPairs(5).
		SetOwnerScore(50).
		SetVisitorScore(15).
		SetOwnerForfeits(1).
		SetVisitorForfeits(2).
		SetLastVisitorId(3003).
		SetTieCooldownUntil(until).
		SetGameType(record.GameTypeOmok).
		Build()

	if r.Title() != "Room Title" {
		t.Errorf("Title() = %q", r.Title())
	}
	if !r.Private() {
		t.Errorf("Private() = false, want true")
	}
	if r.Password() != "secret" {
		t.Errorf("Password() = %q", r.Password())
	}
	if r.PieceType() != 3 {
		t.Errorf("PieceType() = %d", r.PieceType())
	}
	if r.VisitorId() != 3003 {
		t.Errorf("VisitorId() = %d", r.VisitorId())
	}
	if !r.VisitorReady() {
		t.Errorf("VisitorReady() = false, want true")
	}
	if !r.InProgress() {
		t.Errorf("InProgress() = false, want true")
	}
	if !r.DeniedTie(0) || !r.DeniedTie(1) {
		t.Errorf("DeniedTie(0/1) = %v/%v, want true/true", r.DeniedTie(0), r.DeniedTie(1))
	}
	if !r.ExitAfter(0) || !r.ExitAfter(1) {
		t.Errorf("ExitAfter(0/1) = %v/%v, want true/true", r.ExitAfter(0), r.ExitAfter(1))
	}
	if r.FirstMover() != 0 {
		t.Errorf("FirstMover() = %d, want 0", r.FirstMover())
	}
	if r.CurrentTurn() != 1 {
		t.Errorf("CurrentTurn() = %d, want 1", r.CurrentTurn())
	}
	if r.Board() != board {
		t.Errorf("Board() mismatch")
	}
	if len(r.Moves()) != 1 || r.Moves()[0] != (Move{X: 1, Y: 2, Stone: 1}) {
		t.Errorf("Moves() = %v", r.Moves())
	}
	if len(r.Deck()) != 4 {
		t.Errorf("Deck() = %v", r.Deck())
	}
	if r.FirstSlot() != 1 {
		t.Errorf("FirstSlot() = %d, want 1", r.FirstSlot())
	}
	if r.OwnerPairs() != 4 || r.VisitorPairs() != 5 {
		t.Errorf("OwnerPairs/VisitorPairs = %d/%d", r.OwnerPairs(), r.VisitorPairs())
	}
	if r.OwnerScore() != 50 || r.VisitorScore() != 15 {
		t.Errorf("OwnerScore/VisitorScore = %d/%d", r.OwnerScore(), r.VisitorScore())
	}
	if r.OwnerForfeits() != 1 || r.VisitorForfeits() != 2 {
		t.Errorf("OwnerForfeits/VisitorForfeits = %d/%d", r.OwnerForfeits(), r.VisitorForfeits())
	}
	if r.LastVisitorId() != 3003 {
		t.Errorf("LastVisitorId() = %d", r.LastVisitorId())
	}
	if !r.TieCooldownUntil().Equal(until) {
		t.Errorf("TieCooldownUntil() = %v, want %v", r.TieCooldownUntil(), until)
	}
	if r.GameType() != record.GameTypeOmok {
		t.Errorf("GameType() = %v, want %v", r.GameType(), record.GameTypeOmok)
	}
	if r.Field() != f {
		t.Errorf("Field() mismatch")
	}
}

func TestClone_PreservesFieldsAndAllowsOverride(t *testing.T) {
	f := testField()
	orig := NewBuilder(1, 1001, f).
		SetVisitorId(2002).
		SetTitle("orig").
		Build()

	cloned := Clone(orig).SetTitle("updated").Build()

	if cloned.OwnerId() != orig.OwnerId() {
		t.Errorf("Clone lost OwnerId: got %d want %d", cloned.OwnerId(), orig.OwnerId())
	}
	if cloned.VisitorId() != orig.VisitorId() {
		t.Errorf("Clone lost VisitorId: got %d want %d", cloned.VisitorId(), orig.VisitorId())
	}
	if cloned.Title() != "updated" {
		t.Errorf("Clone override did not apply: Title() = %q", cloned.Title())
	}
	if orig.Title() != "orig" {
		t.Errorf("Clone mutated the original room: Title() = %q", orig.Title())
	}
}

func TestSlotOf_And_OpponentOf(t *testing.T) {
	r := NewBuilder(1, 1001, testField()).SetVisitorId(2002).Build()

	slot, ok := r.SlotOf(1001)
	if !ok || slot != 0 {
		t.Errorf("SlotOf(owner) = (%d, %v), want (0, true)", slot, ok)
	}
	slot, ok = r.SlotOf(2002)
	if !ok || slot != 1 {
		t.Errorf("SlotOf(visitor) = (%d, %v), want (1, true)", slot, ok)
	}
	_, ok = r.SlotOf(9999)
	if ok {
		t.Errorf("SlotOf(stranger) ok = true, want false")
	}

	if got := r.OpponentOf(1001); got != 2002 {
		t.Errorf("OpponentOf(owner) = %d, want 2002", got)
	}
	if got := r.OpponentOf(2002); got != 1001 {
		t.Errorf("OpponentOf(visitor) = %d, want 1001", got)
	}
	if got := r.OpponentOf(9999); got != 0 {
		t.Errorf("OpponentOf(stranger) = %d, want 0", got)
	}
}

func TestMovesAndDeck_DefensiveCopies(t *testing.T) {
	moves := []Move{{X: 1, Y: 2, Stone: 1}}
	deck := []uint32{7, 7, 8, 8}

	r := NewBuilder(2, 1001, testField()).
		SetMoves(moves).
		SetDeck(deck).
		Build()

	// Mutating the slices the caller passed in must not affect the room.
	moves[0] = Move{X: 9, Y: 9, Stone: 9}
	deck[0] = 999
	if got := r.Moves()[0]; got != (Move{X: 1, Y: 2, Stone: 1}) {
		t.Errorf("Room shares SetMoves input: Moves()[0] = %v", got)
	}
	if got := r.Deck()[0]; got != 7 {
		t.Errorf("Room shares SetDeck input: Deck()[0] = %d", got)
	}

	// Mutating the slices returned by the getters must not affect the room.
	r.Moves()[0] = Move{X: 8, Y: 8, Stone: 8}
	r.Deck()[0] = 888
	if got := r.Moves()[0]; got != (Move{X: 1, Y: 2, Stone: 1}) {
		t.Errorf("Moves() shares internal slice: Moves()[0] = %v", got)
	}
	if got := r.Deck()[0]; got != 7 {
		t.Errorf("Deck() shares internal slice: Deck()[0] = %d", got)
	}

	// Nil round-trips as nil.
	empty := NewBuilder(2, 1002, testField()).SetMoves(nil).SetDeck(nil).Build()
	if empty.Moves() != nil {
		t.Errorf("Moves() = %v, want nil", empty.Moves())
	}
	if empty.Deck() != nil {
		t.Errorf("Deck() = %v, want nil", empty.Deck())
	}
}

func TestSlotOf_EmptyVisitor(t *testing.T) {
	r := NewBuilder(1, 1001, testField()).Build()

	_, ok := r.SlotOf(0)
	if ok {
		t.Errorf("SlotOf(0) ok = true, want false")
	}
	if got := r.OpponentOf(1001); got != 0 {
		t.Errorf("OpponentOf(owner) with no visitor = %d, want 0", got)
	}
}
