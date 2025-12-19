package character

import (
	"github.com/Chronicle20/atlas-tenant"
)

type MapKey struct {
	Tenant    tenant.Model
	WorldId   byte
	ChannelId byte
	MapId     uint32
}
