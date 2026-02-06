package saved_location

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
)

type modelBuilder struct {
	id           uuid.UUID
	characterId  uint32
	locationType string
	mapId        _map.Id
	portalId     uint32
}

func NewBuilder() *modelBuilder {
	return &modelBuilder{}
}

func (b *modelBuilder) SetId(id uuid.UUID) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetCharacterId(characterId uint32) *modelBuilder {
	b.characterId = characterId
	return b
}

func (b *modelBuilder) SetLocationType(locationType string) *modelBuilder {
	b.locationType = locationType
	return b
}

func (b *modelBuilder) SetMapId(mapId _map.Id) *modelBuilder {
	b.mapId = mapId
	return b
}

func (b *modelBuilder) SetPortalId(portalId uint32) *modelBuilder {
	b.portalId = portalId
	return b
}

func (b *modelBuilder) Build() Model {
	return Model{
		id:           b.id,
		characterId:  b.characterId,
		locationType: b.locationType,
		mapId:        b.mapId,
		portalId:     b.portalId,
	}
}
