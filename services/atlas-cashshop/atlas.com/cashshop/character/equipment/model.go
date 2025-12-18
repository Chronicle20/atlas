package equipment

import (
	"atlas-cashshop/character/equipment/slot"
	slot2 "github.com/Chronicle20/atlas-constants/inventory/slot"
)

type Model struct {
	slots map[slot2.Type]slot.Model
}

func NewModel() Model {
	m := Model{
		slots: make(map[slot2.Type]slot.Model),
	}
	for _, s := range slot2.Slots {
		m.slots[s.Type] = slot.Model{Position: s.Position}
	}
	return m
}

func (m Model) Get(slotType slot2.Type) (slot.Model, bool) {
	val, ok := m.slots[slotType]
	return val, ok
}

func (m *Model) Set(slotType slot2.Type, val slot.Model) {
	m.slots[slotType] = val
}

func (m Model) Slots() map[slot2.Type]slot.Model {
	return m.slots
}
