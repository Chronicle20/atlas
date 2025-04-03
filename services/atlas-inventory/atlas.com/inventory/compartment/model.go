package compartment

import (
	"atlas-inventory/asset"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
)

type Model struct {
	id            uuid.UUID
	inventoryType inventory.Type
	capacity      uint32
	assets        []asset.Model
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) Type() inventory.Type {
	return m.inventoryType
}

func (m Model) Capacity() uint32 {
	return m.capacity
}

func (m Model) Assets() []asset.Model {
	return m.assets
}

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:            m.id,
		inventoryType: m.inventoryType,
		capacity:      m.capacity,
		assets:        m.assets,
	}
}

type ModelBuilder struct {
	id            uuid.UUID
	inventoryType inventory.Type
	capacity      uint32
	assets        []asset.Model
}

func NewBuilder(id uuid.UUID, it inventory.Type, capacity uint32) *ModelBuilder {
	return &ModelBuilder{
		id:            id,
		inventoryType: it,
		capacity:      capacity,
		assets:        make([]asset.Model, 0),
	}
}

func (b *ModelBuilder) SetCapacity(capacity uint32) *ModelBuilder {
	b.capacity = capacity
	return b
}

func (b *ModelBuilder) AddAsset(a asset.Model) *ModelBuilder {
	b.assets = append(b.assets, a)
	return b
}

func (b *ModelBuilder) SetAssets(as []asset.Model) *ModelBuilder {
	b.assets = as
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		id:            b.id,
		inventoryType: b.inventoryType,
		capacity:      b.capacity,
		assets:        b.assets,
	}
}
