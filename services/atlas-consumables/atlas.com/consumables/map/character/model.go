package character

import (
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type MapKey struct {
	Tenant    tenant.Model
	WorldId   byte
	ChannelId byte
	MapId     uint32
	Instance  uuid.UUID
}
