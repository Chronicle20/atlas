package monster

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type MapKey struct {
	Tenant    tenant.Model
	WorldId   world.Id
	ChannelId channel.Id
	MapId     _map.Id
	Instance  uuid.UUID
}

func NewMapKey(tenant tenant.Model, f field.Model) MapKey {
	return MapKey{
		Tenant:    tenant,
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
	}
}

type MonsterKey struct {
	Tenant    tenant.Model
	MonsterId uint32
}
