package map_

import _map "github.com/Chronicle20/atlas/libs/atlas-constants/map"

// Portal is an immutable model for a single portal in a map.
type Portal struct {
	id          uint32
	name        string
	portalType  uint8
	x           int16
	y           int16
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

func (p Portal) X() int16 {
	return p.x
}

func (p Portal) Y() int16 {
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

// Builder constructs an immutable Model.
type Builder struct {
	id                _map.Id
	returnMapId       _map.Id
	forcedReturnMapId _map.Id
	town              bool
	fieldLimit        uint32
	portals           []Portal
}

func NewBuilder(id _map.Id) *Builder {
	return &Builder{id: id}
}

func (b *Builder) SetReturnMapId(v _map.Id) *Builder {
	b.returnMapId = v
	return b
}

func (b *Builder) SetForcedReturnMapId(v _map.Id) *Builder {
	b.forcedReturnMapId = v
	return b
}

func (b *Builder) SetTown(v bool) *Builder {
	b.town = v
	return b
}

func (b *Builder) SetFieldLimit(v uint32) *Builder {
	b.fieldLimit = v
	return b
}

func (b *Builder) SetPortals(v []Portal) *Builder {
	b.portals = v
	return b
}

func (b *Builder) Build() Model {
	return Model{
		id:                b.id,
		returnMapId:       b.returnMapId,
		forcedReturnMapId: b.forcedReturnMapId,
		town:              b.town,
		fieldLimit:        b.fieldLimit,
		portals:           b.portals,
	}
}
