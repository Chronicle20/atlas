package expression

import (
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type Model struct {
	tenant      tenant.Model
	characterId uint32
	worldId     world.Id
	channelId   channel.Id
	mapId       _map.Id
	instance    uuid.UUID
	expression  uint32
	expiration  time.Time
}

func (m Model) Expiration() time.Time {
	return m.expiration
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) MapId() _map.Id {
	return m.mapId
}

func (m Model) Expression() uint32 {
	return m.expression
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) ChannelId() channel.Id {
	return m.channelId
}

func (m Model) Instance() uuid.UUID {
	return m.instance
}
