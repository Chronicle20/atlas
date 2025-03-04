package equipment

import (
	"atlas-character/equipable"
	"atlas-character/equipment/slot"
)

type Model struct {
	slots map[string]slot.Model
}

func NewModel() Model {
	m := Model{
		slots: make(map[string]slot.Model),
	}
	for _, t := range slot.Types {
		pos, err := slot.PositionFromType(t)
		if err != nil {
			continue
		}
		m.slots[t] = slot.Model{Position: pos}
	}
	return m
}

func (m Model) Get(slotType string) (slot.Model, bool) {
	val, ok := m.slots[slotType]
	return val, ok
}

func (m *Model) Set(slotType string, val slot.Model) {
	m.slots[slotType] = val
}

func (m *Model) SetEquipable(slotType string, cash bool, val equipable.Model) {
	if cash {
		m.slots[slotType] = m.slots[slotType].SetCashEquipable(&val)
	} else {
		m.slots[slotType] = m.slots[slotType].SetEquipable(&val)
	}
}

func (m Model) Slots() map[string]slot.Model {
	return m.slots
}
