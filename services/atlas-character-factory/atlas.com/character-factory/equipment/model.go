package equipment

import (
	"atlas-character-factory/equipment/slot"
)

type Model struct {
	slots map[slot.Type]slot.Model
}

func NewModel() Model {
	m := Model{
		slots: make(map[slot.Type]slot.Model),
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

func (m Model) Get(slotType slot.Type) (slot.Model, bool) {
	val, ok := m.slots[slotType]
	return val, ok
}

func (m *Model) Set(slotType slot.Type, val slot.Model) {
	m.slots[slotType] = val
}

func (m Model) Slots() map[slot.Type]slot.Model {
	return m.slots
}
