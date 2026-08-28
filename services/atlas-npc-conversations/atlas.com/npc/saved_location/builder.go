package saved_location

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// Builder for constructing Model
type Builder struct {
	characterId  uint32
	locationType string
	mapId        _map.Id
	portalId     uint32
}

// NewBuilder creates a new Builder
func NewBuilder() *Builder {
	return &Builder{}
}

// SetCharacterId sets the character ID
func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

// SetLocationType sets the location type
func (b *Builder) SetLocationType(locationType string) *Builder {
	b.locationType = locationType
	return b
}

// SetMapId sets the map ID
func (b *Builder) SetMapId(mapId _map.Id) *Builder {
	b.mapId = mapId
	return b
}

// SetPortalId sets the portal ID
func (b *Builder) SetPortalId(portalId uint32) *Builder {
	b.portalId = portalId
	return b
}

// Build creates the Model
func (b *Builder) Build() Model {
	return Model{
		characterId:  b.characterId,
		locationType: b.locationType,
		mapId:        b.mapId,
		portalId:     b.portalId,
	}
}
