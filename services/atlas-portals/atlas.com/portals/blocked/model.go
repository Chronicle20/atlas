package blocked

import _map "github.com/Chronicle20/atlas-constants/map"

// Model represents a blocked portal for a character
type Model struct {
	characterId uint32
	mapId       _map.Id
	portalId    uint32
}

// CharacterId returns the character ID
func (m Model) CharacterId() uint32 {
	return m.characterId
}

// MapId returns the map ID
func (m Model) MapId() _map.Id {
	return m.mapId
}

// PortalId returns the portal ID
func (m Model) PortalId() uint32 {
	return m.portalId
}

// NewModel creates a new blocked portal model
func NewModel(characterId uint32, mapId _map.Id, portalId uint32) Model {
	return Model{
		characterId: characterId,
		mapId:       mapId,
		portalId:    portalId,
	}
}
