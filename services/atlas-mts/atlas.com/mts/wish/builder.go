package wish

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Builder constructs an immutable wish Model. The id is assigned at create time
// in the administrator, so it is not required here.
type Builder struct {
	id          uuid.UUID
	tenantId    uuid.UUID
	characterId uint32
	itemId      uint32
	createdAt   time.Time
}

func NewBuilder(tenantId uuid.UUID, characterId uint32, itemId uint32) *Builder {
	return &Builder{tenantId: tenantId, characterId: characterId, itemId: itemId}
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

func (b *Builder) SetItemId(itemId uint32) *Builder {
	b.itemId = itemId
	return b
}

func (b *Builder) SetCreatedAt(v time.Time) *Builder {
	b.createdAt = v
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId cannot be nil")
	}
	return Model{
		id:          b.id,
		tenantId:    b.tenantId,
		characterId: b.characterId,
		itemId:      b.itemId,
		createdAt:   b.createdAt,
	}, nil
}
