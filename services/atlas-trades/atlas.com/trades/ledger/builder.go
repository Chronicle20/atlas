package ledger

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Builder is the single construction path for a Model: both the settlement
// handler (which builds an entry to record) and Make (which rebuilds one from
// its persisted rows) go through it. Use it in tests too — this project does
// not use *_testhelpers.go constructors.
type Builder struct {
	id            uuid.UUID
	tenantId      uuid.UUID
	transactionId uuid.UUID
	f             field.Model
	roomType      byte
	settledAt     time.Time
	sides         []SideModel
}

// NewBuilder starts an entry for the given settlement transaction. roomType is
// a miniroom type byte — miniroom.Trade or miniroom.CashTrade. id is allocated
// eagerly so a caller can correlate before the row is written; settledAt
// defaults to now.
func NewBuilder(transactionId uuid.UUID, f field.Model, roomType byte) *Builder {
	return &Builder{
		id:            uuid.New(),
		transactionId: transactionId,
		f:             f,
		roomType:      roomType,
		settledAt:     time.Now(),
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder { b.id = id; return b }

func (b *Builder) SetTenantId(tenantId uuid.UUID) *Builder { b.tenantId = tenantId; return b }

func (b *Builder) SetSettledAt(t time.Time) *Builder { b.settledAt = t; return b }

// AddSide appends one participant's contribution. An entry always ends up with
// exactly two sides (PRD §6); the administrator rejects anything else rather
// than the builder, so a partially-assembled entry is still inspectable.
func (b *Builder) AddSide(characterId character.Id, name string, mesoStaged uint32, mesoTax uint32, mesoDelivered uint32, items []Item) *Builder {
	return b.addSideWithId(uuid.New(), characterId, name, mesoStaged, mesoTax, mesoDelivered, items)
}

// addSideWithId is AddSide with a caller-supplied row id, used by Make to
// preserve the persisted trade_ledger_sides.id.
func (b *Builder) addSideWithId(id uuid.UUID, characterId character.Id, name string, mesoStaged uint32, mesoTax uint32, mesoDelivered uint32, items []Item) *Builder {
	copied := make([]Item, len(items))
	copy(copied, items)
	b.sides = append(b.sides, SideModel{
		id:            id,
		characterId:   characterId,
		characterName: name,
		mesoStaged:    mesoStaged,
		mesoTax:       mesoTax,
		mesoDelivered: mesoDelivered,
		items:         copied,
	})
	return b
}

// Build produces the Model. The side slice is copied so a later AddSide on the
// builder cannot write through into an already-built Model.
func (b *Builder) Build() Model {
	sides := make([]SideModel, len(b.sides))
	copy(sides, b.sides)
	return Model{
		id:            b.id,
		tenantId:      b.tenantId,
		transactionId: b.transactionId,
		f:             b.f,
		roomType:      b.roomType,
		settledAt:     b.settledAt,
		sides:         sides,
	}
}
