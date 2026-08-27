package shop

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func NewBuilder() *Builder {
	return &Builder{}
}

type Builder struct {
	id           uuid.UUID
	characterId  uint32
	shopType     ShopType
	state        State
	title        string
	worldId      world.Id
	channelId    channel.Id
	mapId        uint32
	instanceId   uuid.UUID
	x            int16
	y            int16
	permitItemId uint32
	createdAt    time.Time
	expiresAt    *time.Time
	closedAt     *time.Time
	closeReason  CloseReason
	mesoBalance  uint32
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

func (b *Builder) SetShopType(shopType ShopType) *Builder {
	b.shopType = shopType
	return b
}

func (b *Builder) SetState(state State) *Builder {
	b.state = state
	return b
}

func (b *Builder) SetTitle(title string) *Builder {
	b.title = title
	return b
}

func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.worldId = worldId
	return b
}

func (b *Builder) SetChannelId(channelId channel.Id) *Builder {
	b.channelId = channelId
	return b
}

func (b *Builder) SetMapId(mapId uint32) *Builder {
	b.mapId = mapId
	return b
}

func (b *Builder) SetInstanceId(instanceId uuid.UUID) *Builder {
	b.instanceId = instanceId
	return b
}

func (b *Builder) SetX(x int16) *Builder {
	b.x = x
	return b
}

func (b *Builder) SetY(y int16) *Builder {
	b.y = y
	return b
}

func (b *Builder) SetPermitItemId(permitItemId uint32) *Builder {
	b.permitItemId = permitItemId
	return b
}

func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

func (b *Builder) SetExpiresAt(expiresAt *time.Time) *Builder {
	b.expiresAt = expiresAt
	return b
}

func (b *Builder) SetClosedAt(closedAt *time.Time) *Builder {
	b.closedAt = closedAt
	return b
}

func (b *Builder) SetCloseReason(closeReason CloseReason) *Builder {
	b.closeReason = closeReason
	return b
}

func (b *Builder) SetMesoBalance(mesoBalance uint32) *Builder {
	b.mesoBalance = mesoBalance
	return b
}

func (b *Builder) Build() (Model, error) {
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
		worldId:      b.worldId,
		channelId:    b.channelId,
		mapId:        b.mapId,
		instanceId:   b.instanceId,
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

func Clone(m Model) *Builder {
	return &Builder{
		id:           m.id,
		characterId:  m.characterId,
		shopType:     m.shopType,
		state:        m.state,
		title:        m.title,
		worldId:      m.worldId,
		channelId:    m.channelId,
		mapId:        m.mapId,
		instanceId:   m.instanceId,
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
