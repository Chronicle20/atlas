package saved_location

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
)

type Model struct {
	id           uuid.UUID
	characterId  uint32
	locationType string
	mapId        _map.Id
	portalId     uint32
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) LocationType() string {
	return m.locationType
}

func (m Model) MapId() _map.Id {
	return m.mapId
}

func (m Model) PortalId() uint32 {
	return m.portalId
}
