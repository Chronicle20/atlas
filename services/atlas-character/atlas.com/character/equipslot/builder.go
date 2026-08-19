package equipslot

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type modelBuilder struct {
	id          uuid.UUID
	tenantId    uuid.UUID
	characterId uint32
	slotIndex   int16
	expiresAt   time.Time
}

func NewBuilder() *modelBuilder {
	return &modelBuilder{}
}

func (b *modelBuilder) SetId(id uuid.UUID) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetTenantId(tenantId uuid.UUID) *modelBuilder {
	b.tenantId = tenantId
	return b
}

func (b *modelBuilder) SetCharacterId(characterId uint32) *modelBuilder {
	b.characterId = characterId
	return b
}

func (b *modelBuilder) SetSlotIndex(slotIndex int16) *modelBuilder {
	b.slotIndex = slotIndex
	return b
}

func (b *modelBuilder) SetExpiresAt(expiresAt time.Time) *modelBuilder {
	b.expiresAt = expiresAt
	return b
}

func (b *modelBuilder) Build() (Model, error) {
	if b.id == uuid.Nil {
		return Model{}, errors.New("id is required")
	}
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}
	return Model{
		id:          b.id,
		tenantId:    b.tenantId,
		characterId: b.characterId,
		slotIndex:   b.slotIndex,
		expiresAt:   b.expiresAt,
	}, nil
}
