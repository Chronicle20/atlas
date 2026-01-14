package list

import (
	"atlas-buddies/buddy"
	"errors"

	"github.com/google/uuid"
)

type Builder struct {
	tenantId    uuid.UUID
	id          uuid.UUID
	characterId uint32
	capacity    byte
	buddies     []buddy.Model
}

func NewBuilder(tenantId uuid.UUID, characterId uint32) *Builder {
	return &Builder{
		tenantId:    tenantId,
		characterId: characterId,
		capacity:    20,
		buddies:     []buddy.Model{},
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetCapacity(capacity byte) *Builder {
	b.capacity = capacity
	return b
}

func (b *Builder) SetBuddies(buddies []buddy.Model) *Builder {
	b.buddies = buddies
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId is required")
	}
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}
	if b.capacity == 0 {
		return Model{}, errors.New("capacity must be greater than 0")
	}

	return Model{
		tenantId:    b.tenantId,
		id:          b.id,
		characterId: b.characterId,
		capacity:    b.capacity,
		buddies:     b.buddies,
	}, nil
}
