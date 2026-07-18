package game

import (
	"atlas-mini-games/game/omok"
	"atlas-mini-games/record"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Builder constructs an immutable Room via copy-on-write: it holds a Room
// value and every Set<Field> mutates that copy, never the original a
// Builder was cloned from.
type Builder struct {
	room Room
}

// NewBuilder starts a Builder for a brand-new room. Defaults: FirstMover=1
// (owner moves first) and FirstSlot=-1 (no pending MatchCards flip).
// CurrentTurn is left unset (0) until START.
func NewBuilder(roomType byte, ownerId uint32, f field.Model) *Builder {
	return &Builder{room: Room{
		roomType:   roomType,
		ownerId:    ownerId,
		field:      f,
		firstMover: 1,
		firstSlot:  -1,
	}}
}

// Clone seeds a Builder from an existing Room so callers can apply targeted
// Set<Field> mutations without repeating every field. The source Room is
// never mutated — Clone copies it by value.
func Clone(r Room) *Builder {
	return &Builder{room: r}
}

func (b *Builder) SetRoomType(roomType byte) *Builder {
	b.room.roomType = roomType
	return b
}

func (b *Builder) SetOwnerId(ownerId uint32) *Builder {
	b.room.ownerId = ownerId
	return b
}

func (b *Builder) SetField(f field.Model) *Builder {
	b.room.field = f
	return b
}

func (b *Builder) SetTitle(title string) *Builder {
	b.room.title = title
	return b
}

func (b *Builder) SetPrivate(private bool) *Builder {
	b.room.private = private
	return b
}

func (b *Builder) SetPassword(password string) *Builder {
	b.room.password = password
	return b
}

func (b *Builder) SetPieceType(pieceType byte) *Builder {
	b.room.pieceType = pieceType
	return b
}

func (b *Builder) SetVisitorId(visitorId uint32) *Builder {
	b.room.visitorId = visitorId
	return b
}

func (b *Builder) SetVisitorReady(visitorReady bool) *Builder {
	b.room.visitorReady = visitorReady
	return b
}

func (b *Builder) SetInProgress(inProgress bool) *Builder {
	b.room.inProgress = inProgress
	return b
}

// SetDeniedTie sets the deny-tie flag for slot (0 owner, 1 visitor). Any
// slot other than 0/1 is ignored.
func (b *Builder) SetDeniedTie(slot byte, denied bool) *Builder {
	if slot <= 1 {
		b.room.deniedTie[slot] = denied
	}
	return b
}

// SetExitAfter sets the exit-after-game flag for slot (0 owner, 1 visitor).
// Any slot other than 0/1 is ignored.
func (b *Builder) SetExitAfter(slot byte, exit bool) *Builder {
	if slot <= 1 {
		b.room.exitAfter[slot] = exit
	}
	return b
}

func (b *Builder) SetFirstMover(firstMover byte) *Builder {
	b.room.firstMover = firstMover
	return b
}

func (b *Builder) SetCurrentTurn(currentTurn byte) *Builder {
	b.room.currentTurn = currentTurn
	return b
}

func (b *Builder) SetBoard(board [omok.Cells]byte) *Builder {
	b.room.board = board
	return b
}

// SetMoves stores a defensive copy of moves so a caller retaining the slice
// cannot mutate the built Room (and, once registered, registry state)
// through the shared backing array.
func (b *Builder) SetMoves(moves []Move) *Builder {
	if moves == nil {
		b.room.moves = nil
		return b
	}
	out := make([]Move, len(moves))
	copy(out, moves)
	b.room.moves = out
	return b
}

// SetDeck stores a defensive copy of deck, for the same reason as SetMoves.
func (b *Builder) SetDeck(deck []uint32) *Builder {
	if deck == nil {
		b.room.deck = nil
		return b
	}
	out := make([]uint32, len(deck))
	copy(out, deck)
	b.room.deck = out
	return b
}

func (b *Builder) SetFirstSlot(firstSlot int16) *Builder {
	b.room.firstSlot = firstSlot
	return b
}

func (b *Builder) SetOwnerPairs(ownerPairs byte) *Builder {
	b.room.ownerPairs = ownerPairs
	return b
}

func (b *Builder) SetVisitorPairs(visitorPairs byte) *Builder {
	b.room.visitorPairs = visitorPairs
	return b
}

func (b *Builder) SetOwnerScore(ownerScore int32) *Builder {
	b.room.ownerScore = ownerScore
	return b
}

func (b *Builder) SetVisitorScore(visitorScore int32) *Builder {
	b.room.visitorScore = visitorScore
	return b
}

func (b *Builder) SetOwnerForfeits(ownerForfeits byte) *Builder {
	b.room.ownerForfeits = ownerForfeits
	return b
}

func (b *Builder) SetVisitorForfeits(visitorForfeits byte) *Builder {
	b.room.visitorForfeits = visitorForfeits
	return b
}

func (b *Builder) SetLastVisitorId(lastVisitorId uint32) *Builder {
	b.room.lastVisitorId = lastVisitorId
	return b
}

func (b *Builder) SetTieCooldownUntil(t time.Time) *Builder {
	b.room.tieCooldownUntil = t
	return b
}

func (b *Builder) SetSkipCooldownUntil(t time.Time) *Builder {
	b.room.skipCooldownUntil = t
	return b
}

func (b *Builder) SetGameType(gameType record.GameType) *Builder {
	b.room.gameType = gameType
	return b
}

// Build returns the accumulated Room. Building the same Builder twice
// returns two Room values with identical contents (Room is a value type),
// so mutating the Builder further after a Build call does not retroactively
// change the returned Room.
func (b *Builder) Build() Room {
	return b.room
}
