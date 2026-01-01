package pet

// Model represents a pet in the NPC conversations domain
type Model struct {
	id   uint32
	slot int8
}

// NewModel creates a new pet model
func NewModel(id uint32, slot int8) Model {
	return Model{
		id:   id,
		slot: slot,
	}
}

// Id returns the pet's unique identifier
func (m Model) Id() uint32 {
	return m.id
}

// Slot returns the pet's slot position
func (m Model) Slot() int8 {
	return m.slot
}

// IsSpawned returns true if the pet is currently spawned (slot >= 0)
func (m Model) IsSpawned() bool {
	return m.slot >= 0
}
