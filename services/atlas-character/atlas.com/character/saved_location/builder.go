package saved_location

import (
	"github.com/google/uuid"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type builder struct {
	id           uuid.UUID
	characterId  uint32
	locationType string
	mapId        _map.Id
	portalId     uint32
}

func NewBuilder() *builder {
	return &builder{}
}

func (b *builder) SetId(id uuid.UUID) *builder {
	b.id = id
	return b
}

func (b *builder) SetCharacterId(characterId uint32) *builder {
	b.characterId = characterId
	return b
}

func (b *builder) SetLocationType(locationType string) *builder {
	b.locationType = locationType
	return b
}

func (b *builder) SetMapId(mapId _map.Id) *builder {
	b.mapId = mapId
	return b
}

func (b *builder) SetPortalId(portalId uint32) *builder {
	b.portalId = portalId
	return b
}

func (b *builder) Build() Model {
	return Model{
		id:           b.id,
		characterId:  b.characterId,
		locationType: b.locationType,
		mapId:        b.mapId,
		portalId:     b.portalId,
	}
}
