package compartment

import (
	"atlas-channel/asset"
	"sort"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
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

// FindFirstByItemIdWithQuantity returns the matching asset in the
// lowest-index slot whose quantity is at least `quantity`. Candidates are
// sorted by slot ascending before scanning, so the result is deterministic
// regardless of the backing slice's order (unlike FindFirstByItemId). A slot
// holding less than `quantity` is skipped — single-slot draw only.
func (m Model) FindFirstByItemIdWithQuantity(templateId uint32, quantity int16) (*asset.Model, bool) {
	matching := make([]asset.Model, 0, len(m.Assets()))
	for _, a := range m.Assets() {
		if a.TemplateId() == templateId {
			matching = append(matching, a)
		}
	}
	sort.Slice(matching, func(i, j int) bool { return matching[i].Slot() < matching[j].Slot() })
	for _, a := range matching {
		if int64(a.Quantity()) >= int64(quantity) {
			a := a
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

// FindFirstByClassification returns the first asset in the compartment whose
// template resolves to the given item classification (e.g. Note = 509).
func (m Model) FindFirstByClassification(c item.Classification) (*asset.Model, bool) {
	for _, a := range m.Assets() {
		if item.GetClassification(item.Id(a.TemplateId())) == c {
			return &a, true
		}
	}
	return nil, false
}
