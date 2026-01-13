package fame

import (
	"errors"

	"github.com/google/uuid"
)

type Builder struct {
	tenantId    uuid.UUID
	characterId uint32
	targetId    uint32
	amount      int8
}

func NewBuilder(tenantId uuid.UUID, characterId uint32, targetId uint32, amount int8) *Builder {
	return &Builder{
		tenantId:    tenantId,
		characterId: characterId,
		targetId:    targetId,
		amount:      amount,
	}
}

func (b *Builder) SetTenantId(tenantId uuid.UUID) *Builder {
	b.tenantId = tenantId
	return b
}

func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

func (b *Builder) SetTargetId(targetId uint32) *Builder {
	b.targetId = targetId
	return b
}

func (b *Builder) SetAmount(amount int8) *Builder {
	b.amount = amount
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId is required")
	}
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}
	if b.targetId == 0 {
		return Model{}, errors.New("targetId is required")
	}
	if b.amount != 1 && b.amount != -1 {
		return Model{}, errors.New("amount must be 1 or -1")
	}

	return Model{
		tenantId:    b.tenantId,
		characterId: b.characterId,
		targetId:    b.targetId,
		amount:      b.amount,
	}, nil
}
