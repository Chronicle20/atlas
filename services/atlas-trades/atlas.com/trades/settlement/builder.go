package settlement

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Builder is the single construction path for a Model: both the settlement
// submission and Make (which rebuilds one from its persisted rows) go through
// it. Use it in tests too — this project does not use *_testhelpers.go
// constructors.
type Builder struct {
	id            uuid.UUID
	tenantId      uuid.UUID
	tenantRegion  string
	tenantMajor   uint16
	tenantMinor   uint16
	transactionId uuid.UUID
	roomId        uuid.UUID
	handle        uint32
	roomType      byte
	f             field.Model
	ownerId       character.Id
	visitorId     character.Id
	submittedAt   time.Time
	sides         []SideModel
}

// NewBuilder starts a record for one submitted settlement. id is allocated
// eagerly so a caller can correlate before the row is written; submittedAt
// defaults to now.
func NewBuilder(transactionId uuid.UUID, roomId uuid.UUID, handle uint32, roomType byte, f field.Model, ownerId character.Id, visitorId character.Id) *Builder {
	return &Builder{
		id:            uuid.New(),
		transactionId: transactionId,
		roomId:        roomId,
		handle:        handle,
		roomType:      roomType,
		f:             f,
		ownerId:       ownerId,
		visitorId:     visitorId,
		submittedAt:   time.Now(),
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder { b.id = id; return b }

// SetTenant stamps the tenant this settlement belongs to. The administrator
// sets it on write from the request tenant; Make sets it on read.
func (b *Builder) SetTenant(id uuid.UUID, region string, major uint16, minor uint16) *Builder {
	b.tenantId = id
	b.tenantRegion = region
	b.tenantMajor = major
	b.tenantMinor = minor
	return b
}

func (b *Builder) SetSubmittedAt(t time.Time) *Builder { b.submittedAt = t; return b }

// AddSide appends one participant's contribution. A record always ends up with
// exactly two sides; the administrator rejects anything else rather than the
// builder, so a partially-assembled record is still inspectable.
func (b *Builder) AddSide(position byte, characterId character.Id, name string, mesoStaged uint32, mesoTax uint32, mesoDelivered uint32, items []Item) *Builder {
	return b.addSideWithId(uuid.New(), position, characterId, name, mesoStaged, mesoTax, mesoDelivered, items)
}

// addSideWithId is AddSide with a caller-supplied row id, used by Make to
// preserve the persisted trade_settlement_sides.id.
func (b *Builder) addSideWithId(id uuid.UUID, position byte, characterId character.Id, name string, mesoStaged uint32, mesoTax uint32, mesoDelivered uint32, items []Item) *Builder {
	copied := make([]Item, len(items))
	copy(copied, items)
	b.sides = append(b.sides, SideModel{
		id:            id,
		position:      position,
		characterId:   characterId,
		characterName: name,
		mesoStaged:    mesoStaged,
		mesoTax:       mesoTax,
		mesoDelivered: mesoDelivered,
		items:         copied,
	})
	return b
}

// Build produces the Model, with the sides ordered by seat. The side slice is
// copied so a later AddSide on the builder cannot write through into an
// already-built Model.
func (b *Builder) Build() Model {
	sides := make([]SideModel, len(b.sides))
	copy(sides, b.sides)
	if len(sides) == 2 && sides[0].position > sides[1].position {
		sides[0], sides[1] = sides[1], sides[0]
	}
	return Model{
		id:            b.id,
		tenantId:      b.tenantId,
		tenantRegion:  b.tenantRegion,
		tenantMajor:   b.tenantMajor,
		tenantMinor:   b.tenantMinor,
		transactionId: b.transactionId,
		roomId:        b.roomId,
		handle:        b.handle,
		roomType:      b.roomType,
		f:             b.f,
		ownerId:       b.ownerId,
		visitorId:     b.visitorId,
		submittedAt:   b.submittedAt,
		sides:         sides,
	}
}
