package transaction

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// Builder constructs an immutable transaction Model. The id is assigned at
// create time in the administrator, so it is not required here.
type Builder struct {
	id             uuid.UUID
	tenantId       uuid.UUID
	worldId        world.Id
	characterId    uint32
	counterpartyId uint32
	itemId         uint32
	quantity       uint32
	totalPrice     uint32
	kind           string
	createdAt      time.Time
}

func NewBuilder(tenantId uuid.UUID, worldId world.Id, characterId uint32) *Builder {
	return &Builder{tenantId: tenantId, worldId: worldId, characterId: characterId}
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.worldId = worldId
	return b
}

func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

func (b *Builder) SetCounterpartyId(counterpartyId uint32) *Builder {
	b.counterpartyId = counterpartyId
	return b
}

func (b *Builder) SetItemId(itemId uint32) *Builder {
	b.itemId = itemId
	return b
}

func (b *Builder) SetQuantity(quantity uint32) *Builder {
	b.quantity = quantity
	return b
}

func (b *Builder) SetTotalPrice(totalPrice uint32) *Builder {
	b.totalPrice = totalPrice
	return b
}

func (b *Builder) SetKind(kind string) *Builder {
	b.kind = kind
	return b
}

func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId cannot be nil")
	}
	return Model{
		id:             b.id,
		tenantId:       b.tenantId,
		worldId:        b.worldId,
		characterId:    b.characterId,
		counterpartyId: b.counterpartyId,
		itemId:         b.itemId,
		quantity:       b.quantity,
		totalPrice:     b.totalPrice,
		kind:           b.kind,
		createdAt:      b.createdAt,
	}, nil
}
