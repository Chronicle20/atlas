package monster

import (
	"github.com/Chronicle20/atlas-tenant"
)

type MapKey struct {
	Tenant    tenant.Model
	WorldId   byte
	ChannelId byte
	MapId     uint32
}

func NewMapKey(tenant tenant.Model, worldId byte, channelId byte, mapId uint32) MapKey {
	return MapKey{tenant, worldId, channelId, mapId}
}

type MonsterKey struct {
	Tenant    tenant.Model
	MonsterId uint32
}
