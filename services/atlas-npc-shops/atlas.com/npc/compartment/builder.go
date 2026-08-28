package compartment

import (
	"atlas-npc/asset"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

func Clone(m Model) *Builder {
	return &Builder{
		id:            m.id,
		characterId:   m.characterId,
		inventoryType: m.inventoryType,
		capacity:      m.capacity,
		assets:        m.assets,
	}
}

type Builder struct {
	id            uuid.UUID
	characterId   uint32
	inventoryType inventory.Type
	capacity      uint32
	assets        []asset.Model
}

func NewBuilder(id uuid.UUID, characterId uint32, it inventory.Type, capacity uint32) *Builder {
	return &Builder{
		id:            id,
		characterId:   characterId,
		inventoryType: it,
		capacity:      capacity,
		assets:        make([]asset.Model, 0),
	}
}

func (b *Builder) SetCapacity(capacity uint32) *Builder {
	b.capacity = capacity
	return b
}

func (b *Builder) AddAsset(a asset.Model) *Builder {
	b.assets = append(b.assets, a)
	return b
}

func (b *Builder) SetAssets(as []asset.Model) *Builder {
	b.assets = as
	return b
}

func (b *Builder) Build() Model {
	return Model{
		id:            b.id,
		characterId:   b.characterId,
		inventoryType: b.inventoryType,
		capacity:      b.capacity,
		assets:        b.assets,
	}
}
