// Package game holds the mini-game Room model and its tenant-partitioned
// in-memory registry. Room is a snapshot of one Omok/MatchCards room's
// state; the registry (registry.go) is the only mutable state — every
// mutation goes through Registry.Update, which swaps an old Room for a new
// one built via Clone under a single write lock.
package game

import (
	"time"

	"atlas-mini-games/game/omok"
	"atlas-mini-games/record"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Move records one Omok stone placement, in play order, so a retreat can
// pop the most recent move(s) off the history.
type Move struct {
	X     uint32
	Y     uint32
	Stone byte
}

// Room is the immutable state of one mini-game room (Omok or MatchCards).
// Id() == OwnerId (design D2): a room is keyed by its owner's character id,
// so a character can own at most one room at a time. Rooms are never
// mutated in place — Registry.Update swaps a Room built from Clone(cur).
type Room struct {
	roomType         byte
	ownerId          uint32
	field            field.Model
	title            string
	private          bool
	password         string
	pieceType        byte
	visitorId        uint32
	visitorReady     bool
	inProgress       bool
	deniedTie        [2]bool
	exitAfter        [2]bool
	firstMover       byte
	currentTurn      byte
	board            [omok.Cells]byte
	moves            []Move
	deck             []uint32
	firstSlot        int16
	ownerPairs       byte
	visitorPairs     byte
	ownerScore       int32
	visitorScore     int32
	ownerForfeits    byte
	visitorForfeits  byte
	lastVisitorId    uint32
	tieCooldownUntil time.Time
	gameType         record.GameType
}

// RoomType returns 1 for Omok, 2 for MatchCards.
func (r Room) RoomType() byte {
	return r.roomType
}

func (r Room) OwnerId() uint32 {
	return r.ownerId
}

// Id returns the room's identity, which is always its owner's character id
// (design D2).
func (r Room) Id() uint32 {
	return r.ownerId
}

func (r Room) Field() field.Model {
	return r.field
}

func (r Room) Title() string {
	return r.title
}

func (r Room) Private() bool {
	return r.private
}

func (r Room) Password() string {
	return r.password
}

func (r Room) PieceType() byte {
	return r.pieceType
}

// VisitorId returns 0 when the room has no visitor.
func (r Room) VisitorId() uint32 {
	return r.visitorId
}

func (r Room) VisitorReady() bool {
	return r.visitorReady
}

func (r Room) InProgress() bool {
	return r.inProgress
}

// DeniedTie reports whether the occupant of slot (0 owner, 1 visitor) has
// declined the current tie proposal. Any slot other than 0/1 returns false.
func (r Room) DeniedTie(slot byte) bool {
	if slot > 1 {
		return false
	}
	return r.deniedTie[slot]
}

// ExitAfter reports whether the occupant of slot (0 owner, 1 visitor) has
// requested to leave once the current game ends. Any slot other than 0/1
// returns false.
func (r Room) ExitAfter(slot byte) bool {
	if slot > 1 {
		return false
	}
	return r.exitAfter[slot]
}

// FirstMover is the slot (0 owner, 1 visitor) granted the first move of the
// next game.
func (r Room) FirstMover() byte {
	return r.firstMover
}

// CurrentTurn is the slot (0 owner, 1 visitor) whose move is currently
// accepted. Unset (0) until START.
func (r Room) CurrentTurn() byte {
	return r.currentTurn
}

func (r Room) Board() [omok.Cells]byte {
	return r.board
}

// Moves returns a defensive copy of the move history: Room values share the
// slice's backing array when copied, so handing out the internal slice would
// let a caller mutate registry state outside Registry.Update.
func (r Room) Moves() []Move {
	if r.moves == nil {
		return nil
	}
	out := make([]Move, len(r.moves))
	copy(out, r.moves)
	return out
}

// Deck returns a defensive copy of the deck, for the same reason as Moves.
func (r Room) Deck() []uint32 {
	if r.deck == nil {
		return nil
	}
	out := make([]uint32, len(r.deck))
	copy(out, r.deck)
	return out
}

// FirstSlot is the deck index of the pending MatchCards first-flip card
// (the "slot" in the mode-68 wire sense: a card position, not a player
// slot); -1 means no pending flip. The second flip compares the card at
// this index against the newly flipped one.
func (r Room) FirstSlot() int16 {
	return r.firstSlot
}

func (r Room) OwnerPairs() byte {
	return r.ownerPairs
}

func (r Room) VisitorPairs() byte {
	return r.visitorPairs
}

func (r Room) OwnerScore() int32 {
	return r.ownerScore
}

func (r Room) VisitorScore() int32 {
	return r.visitorScore
}

func (r Room) OwnerForfeits() byte {
	return r.ownerForfeits
}

func (r Room) VisitorForfeits() byte {
	return r.visitorForfeits
}

// LastVisitorId is the most recent non-zero VisitorId, retained after a
// visitor leaves (rematch/record bookkeeping).
func (r Room) LastVisitorId() uint32 {
	return r.lastVisitorId
}

// TieCooldownUntil is the point in time before which a tie result is not
// eligible for the tie-score bonus (5-minute cooldown, design).
func (r Room) TieCooldownUntil() time.Time {
	return r.tieCooldownUntil
}

func (r Room) GameType() record.GameType {
	return r.gameType
}

// SlotOf returns the slot (0 owner, 1 visitor) occupied by characterId, or
// (0, false) if characterId is neither the current owner nor the current
// visitor (including characterId == 0, which never matches a slot).
func (r Room) SlotOf(characterId uint32) (byte, bool) {
	if characterId == 0 {
		return 0, false
	}
	if characterId == r.ownerId {
		return 0, true
	}
	if r.visitorId != 0 && characterId == r.visitorId {
		return 1, true
	}
	return 0, false
}

// OpponentOf returns the character id occupying the slot opposite
// characterId, or 0 when characterId is not a member of the room or the
// opposite slot is empty.
func (r Room) OpponentOf(characterId uint32) uint32 {
	slot, ok := r.SlotOf(characterId)
	if !ok {
		return 0
	}
	if slot == 0 {
		return r.visitorId
	}
	return r.ownerId
}
