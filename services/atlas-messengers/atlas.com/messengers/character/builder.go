package character

import (
	"errors"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

var ErrMissingId = errors.New("character id is required")
var ErrMissingName = errors.New("character name is required")

type builder struct {
	tenantId    uuid.UUID
	id          uint32
	name        string
	worldId     world.Id
	channelId   channel.Id
	messengerId uint32
	online      bool
}

func NewBuilder() *builder {
	return &builder{}
}

func (b *builder) SetTenantId(tenantId uuid.UUID) *builder {
	b.tenantId = tenantId
	return b
}

func (b *builder) SetId(id uint32) *builder {
	b.id = id
	return b
}

func (b *builder) SetName(name string) *builder {
	b.name = name
	return b
}

func (b *builder) SetWorldId(worldId world.Id) *builder {
	b.worldId = worldId
	return b
}

func (b *builder) SetChannelId(channelId channel.Id) *builder {
	b.channelId = channelId
	return b
}

func (b *builder) SetMessengerId(messengerId uint32) *builder {
	b.messengerId = messengerId
	return b
}

func (b *builder) SetOnline(online bool) *builder {
	b.online = online
	return b
}

func (b *builder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, ErrMissingId
	}
	if b.name == "" {
		return Model{}, ErrMissingName
	}
	return Model{
		tenantId:    b.tenantId,
		id:          b.id,
		name:        b.name,
		ch:          channel.NewModel(b.worldId, b.channelId),
		messengerId: b.messengerId,
		online:      b.online,
	}, nil
}
