package map_

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
)

// Portal is an immutable model for a single portal in a map.
type Portal struct {
	id          uint32
	name        string
	portalType  uint8
	x           point.X
	y           point.Y
	targetMapId _map.Id
}

func (p Portal) Id() uint32 {
	return p.id
}

func (p Portal) Name() string {
	return p.name
}

func (p Portal) Type() uint8 {
	return p.portalType
}

func (p Portal) X() point.X {
	return p.x
}

func (p Portal) Y() point.Y {
	return p.y
}

func (p Portal) TargetMapId() _map.Id {
	return p.targetMapId
}

// Model is an immutable model for a map returned by atlas-data.
type Model struct {
	id                _map.Id
	returnMapId       _map.Id
	forcedReturnMapId _map.Id
	town              bool
	fieldLimit        uint32
	portals           []Portal
}

func (m Model) Id() _map.Id {
	return m.id
}

func (m Model) ReturnMapId() _map.Id {
	return m.returnMapId
}

func (m Model) ForcedReturnMapId() _map.Id {
	return m.forcedReturnMapId
}

func (m Model) Town() bool {
	return m.town
}

func (m Model) FieldLimit() uint32 {
	return m.fieldLimit
}

func (m Model) Portals() []Portal {
	return m.portals
}
