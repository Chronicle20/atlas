package shop

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

func NewBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

type ModelBuilder struct {
	id           uuid.UUID
	characterId  uint32
	shopType     ShopType
	state        State
	title        string
	mapId        uint32
	x            int16
	y            int16
	permitItemId uint32
	createdAt    time.Time
	expiresAt    *time.Time
	closedAt     *time.Time
	closeReason  CloseReason
	mesoBalance  uint32
}

func (b *ModelBuilder) SetId(id uuid.UUID) *ModelBuilder {
	b.id = id
	return b
}

func (b *ModelBuilder) SetCharacterId(characterId uint32) *ModelBuilder {
	b.characterId = characterId
	return b
}

func (b *ModelBuilder) SetShopType(shopType ShopType) *ModelBuilder {
	b.shopType = shopType
	return b
}

func (b *ModelBuilder) SetState(state State) *ModelBuilder {
	b.state = state
	return b
}

func (b *ModelBuilder) SetTitle(title string) *ModelBuilder {
	b.title = title
	return b
}

func (b *ModelBuilder) SetMapId(mapId uint32) *ModelBuilder {
	b.mapId = mapId
	return b
}

func (b *ModelBuilder) SetX(x int16) *ModelBuilder {
	b.x = x
	return b
}

func (b *ModelBuilder) SetY(y int16) *ModelBuilder {
	b.y = y
	return b
}

func (b *ModelBuilder) SetPermitItemId(permitItemId uint32) *ModelBuilder {
	b.permitItemId = permitItemId
	return b
}

func (b *ModelBuilder) SetCreatedAt(createdAt time.Time) *ModelBuilder {
	b.createdAt = createdAt
	return b
}

func (b *ModelBuilder) SetExpiresAt(expiresAt *time.Time) *ModelBuilder {
	b.expiresAt = expiresAt
	return b
}

func (b *ModelBuilder) SetClosedAt(closedAt *time.Time) *ModelBuilder {
	b.closedAt = closedAt
	return b
}

func (b *ModelBuilder) SetCloseReason(closeReason CloseReason) *ModelBuilder {
	b.closeReason = closeReason
	return b
}

func (b *ModelBuilder) SetMesoBalance(mesoBalance uint32) *ModelBuilder {
	b.mesoBalance = mesoBalance
	return b
}

func (b *ModelBuilder) Build() (Model, error) {
	if b.id == uuid.Nil {
		return Model{}, errors.New("id is required")
	}
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}
	if b.shopType == 0 {
		return Model{}, errors.New("shopType is required")
	}
	if b.state == 0 {
		return Model{}, errors.New("state is required")
	}
	return Model{
		id:           b.id,
		characterId:  b.characterId,
		shopType:     b.shopType,
		state:        b.state,
		title:        b.title,
		mapId:        b.mapId,
		x:            b.x,
		y:            b.y,
		permitItemId: b.permitItemId,
		createdAt:    b.createdAt,
		expiresAt:    b.expiresAt,
		closedAt:     b.closedAt,
		closeReason:  b.closeReason,
		mesoBalance:  b.mesoBalance,
	}, nil
}

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:           m.id,
		characterId:  m.characterId,
		shopType:     m.shopType,
		state:        m.state,
		title:        m.title,
		mapId:        m.mapId,
		x:            m.x,
		y:            m.y,
		permitItemId: m.permitItemId,
		createdAt:    m.createdAt,
		expiresAt:    m.expiresAt,
		closedAt:     m.closedAt,
		closeReason:  m.closeReason,
		mesoBalance:  m.mesoBalance,
	}
}
