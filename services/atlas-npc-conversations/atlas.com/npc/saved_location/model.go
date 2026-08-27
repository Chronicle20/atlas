package saved_location

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// Model represents a saved location
type Model struct {
	characterId  uint32
	locationType string
	mapId        _map.Id
	portalId     uint32
}

// CharacterId returns the character ID
func (m Model) CharacterId() uint32 {
	return m.characterId
}

// LocationType returns the location type
func (m Model) LocationType() string {
	return m.locationType
}

// MapId returns the map ID
func (m Model) MapId() _map.Id {
	return m.mapId
}

// PortalId returns the portal ID
func (m Model) PortalId() uint32 {
	return m.portalId
}
