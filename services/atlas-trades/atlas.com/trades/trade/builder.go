package trade

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Builder constructs a Room. Use it in tests too — this project does not use
// *_testhelpers.go constructors.
type Builder struct {
	id           uuid.UUID
	handle       uint32
	roomType     byte
	f            field.Model
	state        State
	participants []Participant
	createdAt    time.Time
}

// NewBuilder starts a solo room owned by ownerId at position 0. roomType is a
// miniroom type byte — miniroom.Trade or miniroom.CashTrade. handle defaults to
// ownerId (design §2.3); override with SetHandle only in tests.
func NewBuilder(roomType byte, ownerId character.Id, ownerName string, f field.Model) *Builder {
	return &Builder{
		id:       uuid.New(),
		handle:   uint32(ownerId),
		roomType: roomType,
		f:        f,
		state:    StateOpenSolo,
		participants: []Participant{
			{characterId: ownerId, name: ownerName, position: 0, items: []StagedItem{}},
		},
		createdAt: time.Now(),
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder { b.id = id; return b }

func (b *Builder) SetHandle(h uint32) *Builder { b.handle = h; return b }

func (b *Builder) SetState(s State) *Builder { b.state = s; return b }

func (b *Builder) SetCreatedAt(t time.Time) *Builder { b.createdAt = t; return b }

// SetVisitor seats the invited character at position 1.
func (b *Builder) SetVisitor(characterId character.Id, name string) *Builder {
	b.participants = append(b.participants, Participant{
		characterId: characterId, name: name, position: 1, items: []StagedItem{},
	})
	return b
}

// Build produces the Room. The participant slice is copied so a later Set on
// the builder cannot write through into an already-built Room.
func (b *Builder) Build() Room {
	participants := make([]Participant, len(b.participants))
	copy(participants, b.participants)
	return Room{
		id:           b.id,
		handle:       b.handle,
		roomType:     b.roomType,
		f:            b.f,
		state:        b.state,
		participants: participants,
		createdAt:    b.createdAt,
	}
}
