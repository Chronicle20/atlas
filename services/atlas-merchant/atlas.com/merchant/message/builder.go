package message

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

func NewBuilder() *Builder {
	return &Builder{}
}

type Builder struct {
	id          uuid.UUID
	shopId      uuid.UUID
	characterId uint32
	content     string
	sentAt      time.Time
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetShopId(shopId uuid.UUID) *Builder {
	b.shopId = shopId
	return b
}

func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

func (b *Builder) SetContent(content string) *Builder {
	b.content = content
	return b
}

func (b *Builder) SetSentAt(sentAt time.Time) *Builder {
	b.sentAt = sentAt
	return b
}

func (b *Builder) Build() (Model, error) {
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
