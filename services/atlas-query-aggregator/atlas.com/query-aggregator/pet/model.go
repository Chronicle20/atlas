package pet

// Model represents a pet in the query aggregator
type Model struct {
	id   uint32
	slot int8
}

func NewModel(id uint32, slot int8) Model {
	return Model{
		id:   id,
		slot: slot,
	}
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Slot() int8 {
	return m.slot
}

// IsSpawned returns true if the pet is currently spawned (slot >= 0)
func (m Model) IsSpawned() bool {
	return m.slot >= 0
}
