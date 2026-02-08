package compartment

import (
	"atlas-channel/asset"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
)

type Model struct {
	id            uuid.UUID
	characterId   uint32
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

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) FindBySlot(slot int16) (*asset.Model, bool) {
	for _, a := range m.Assets() {
		if a.Slot() == slot {
			return &a, true
		}
	}
	return nil, false
}

func (m Model) FindById(id uint32) (*asset.Model, bool) {
	for _, a := range m.Assets() {
		if a.Id() == id {
			return &a, true
		}
	}
	return nil, false
}

func (m Model) FindFirstByItemId(templateId uint32) (*asset.Model, bool) {
	for _, a := range m.Assets() {
		if a.TemplateId() == templateId {
			return &a, true
		}
	}
	return nil, false
}

func (m Model) FindByPetId(petId uint32) (*asset.Model, bool) {
	for _, a := range m.Assets() {
		if a.PetId() == petId {
			return &a, true
		}
	}
	return nil, false
}
