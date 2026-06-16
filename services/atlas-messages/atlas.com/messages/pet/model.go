package pet

// Model represents a pet owned by a character.
type Model struct {
	id   uint32
	slot int8
	name string
}

func NewModel(id uint32, slot int8, name string) Model {
	return Model{
		id:   id,
		slot: slot,
		name: name,
	}
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Slot() int8 {
	return m.slot
}

func (m Model) Name() string {
	return m.name
}

// IsSpawned returns true if the pet is currently spawned (slot >= 0).
func (m Model) IsSpawned() bool {
	return m.slot >= 0
}
