package message

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

func NewBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

type ModelBuilder struct {
	id          uuid.UUID
	shopId      uuid.UUID
	characterId uint32
	content     string
	sentAt      time.Time
}

func (b *ModelBuilder) SetId(id uuid.UUID) *ModelBuilder {
	b.id = id
	return b
}

func (b *ModelBuilder) SetShopId(shopId uuid.UUID) *ModelBuilder {
	b.shopId = shopId
	return b
}

func (b *ModelBuilder) SetCharacterId(characterId uint32) *ModelBuilder {
	b.characterId = characterId
	return b
}

func (b *ModelBuilder) SetContent(content string) *ModelBuilder {
	b.content = content
	return b
}

func (b *ModelBuilder) SetSentAt(sentAt time.Time) *ModelBuilder {
	b.sentAt = sentAt
	return b
}

func (b *ModelBuilder) Build() (Model, error) {
	if b.id == uuid.Nil {
		return Model{}, errors.New("id is required")
	}
	if b.shopId == uuid.Nil {
		return Model{}, errors.New("shopId is required")
	}
	return Model{
		id:          b.id,
		shopId:      b.shopId,
		characterId: b.characterId,
		content:     b.content,
		sentAt:      b.sentAt,
	}, nil
}
