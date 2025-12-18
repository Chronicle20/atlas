package compartment

import (
	"atlas-saga-orchestrator/asset"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
)

type Model struct {
	id            uuid.UUID
	characterId   uint32
	inventoryType inventory.Type
	capacity      uint32
	assets        []asset.Model[any]
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

func (m Model) Assets() []asset.Model[any] {
	return m.assets
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) FindBySlot(slot int16) (*asset.Model[any], bool) {
	for _, a := range m.Assets() {
		if a.Slot() == slot {
			return &a, true
		}
	}
	return nil, false
}

func (m Model) FindFirstByItemId(templateId uint32) (*asset.Model[any], bool) {
	for _, a := range m.Assets() {
		if a.TemplateId() == templateId {
			return &a, true
		}
	}
	return nil, false
}

func (m Model) FindByReferenceId(referenceId uint32) (*asset.Model[any], bool) {
	for _, a := range m.Assets() {
		if a.ReferenceId() == referenceId {
			return &a, true
		}
	}
	return nil, false
}

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:            m.id,
		characterId:   m.characterId,
		inventoryType: m.inventoryType,
		capacity:      m.capacity,
		assets:        m.assets,
	}
}

type ModelBuilder struct {
	id            uuid.UUID
	characterId   uint32
	inventoryType inventory.Type
	capacity      uint32
	assets        []asset.Model[any]
}

func NewBuilder(id uuid.UUID, characterId uint32, it inventory.Type, capacity uint32) *ModelBuilder {
	return &ModelBuilder{
		id:            id,
		characterId:   characterId,
		inventoryType: it,
		capacity:      capacity,
		assets:        make([]asset.Model[any], 0),
	}
}

func (b *ModelBuilder) SetCapacity(capacity uint32) *ModelBuilder {
	b.capacity = capacity
	return b
}

func (b *ModelBuilder) AddAsset(a asset.Model[any]) *ModelBuilder {
	b.assets = append(b.assets, a)
	return b
}

func (b *ModelBuilder) SetAssets(as []asset.Model[any]) *ModelBuilder {
	b.assets = as
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		id:            b.id,
		characterId:   b.characterId,
		inventoryType: b.inventoryType,
		capacity:      b.capacity,
		assets:        b.assets,
	}
}
