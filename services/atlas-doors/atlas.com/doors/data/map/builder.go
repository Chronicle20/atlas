package map_

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

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
