package expression

import (
	"github.com/Chronicle20/atlas-tenant"
	"time"
)

type Model struct {
	tenant      tenant.Model
	characterId uint32
	worldId     byte
	channelId   byte
	mapId       uint32
	expression  uint32
	expiration  time.Time
}

func (m Model) Expiration() time.Time {
	return m.expiration
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) MapId() uint32 {
	return m.mapId
}

func (m Model) Expression() uint32 {
	return m.expression
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}

func (m Model) WorldId() byte {
	return m.worldId
}

func (m Model) ChannelId() byte {
	return m.channelId
}
