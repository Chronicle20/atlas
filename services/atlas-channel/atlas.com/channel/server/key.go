package server

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// Key uniquely identifies a per-(tenant, world, channel) server entry in
// the registry. Comparable by value so it can be used as a map key.
// Composed of shared atlas-constants types per DOM-21 (no new ID types).
type Key struct {
	TenantId  uuid.UUID
	WorldId   world.Id
	ChannelId channel.Id
}

// KeyOf builds a Key from a Model.
func KeyOf(m Model) Key {
	t := m.Tenant() // tenant.Model methods take a pointer receiver
	return Key{
		TenantId:  t.Id(),
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
	}
}
