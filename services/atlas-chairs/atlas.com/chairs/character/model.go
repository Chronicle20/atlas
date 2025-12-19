package character

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

type MapKey struct {
	Tenant    tenant.Model
	WorldId   world.Id
	ChannelId channel.Id
	MapId     _map.Id
}
