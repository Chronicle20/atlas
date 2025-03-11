package equipment

import (
	"atlas-character/equipable"
	"atlas-character/equipment/slot"
)

type Model struct {
	slots map[slot.Type]slot.Model
}

func NewModel() Model {
	m := Model{
		slots: make(map[slot.Type]slot.Model),
	}
	for _, s := range slot.Slots {
		m.slots[s.Type] = slot.Model{Position: s.Position}
	}
	return m
}

func (m Model) Get(slotType slot.Type) (slot.Model, bool) {
	val, ok := m.slots[slotType]
	return val, ok
}

func (m *Model) Set(slotType slot.Type, val slot.Model) {
	m.slots[slotType] = val
}

func (m *Model) SetEquipable(slotType slot.Type, cash bool, val equipable.Model) {
	if cash {
		m.slots[slotType] = m.slots[slotType].SetCashEquipable(&val)
	} else {
		m.slots[slotType] = m.slots[slotType].SetEquipable(&val)
	}
}

func (m Model) Slots() map[slot.Type]slot.Model {
	return m.slots
}
