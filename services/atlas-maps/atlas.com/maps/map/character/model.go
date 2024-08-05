package character

import (
	"atlas-maps/tenant"
)

type MapKey struct {
	Tenant    tenant.Model
	WorldId   byte
	ChannelId byte
	MapId     uint32
}
