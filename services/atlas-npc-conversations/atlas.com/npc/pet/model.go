package pet

// Model represents a pet in the NPC conversations domain
type Model struct {
	id         uint32
	templateId uint32
	name       string
	level      byte
	slot       int8
}

// NewModel creates a new pet model
func NewModel(id uint32, templateId uint32, name string, level byte, slot int8) Model {
	return Model{
		id:         id,
		templateId: templateId,
		name:       name,
		level:      level,
		slot:       slot,
	}
}

// Id returns the pet's unique identifier
func (m Model) Id() uint32 {
	return m.id
}

// Name returns the pet's given name
func (m Model) Name() string {
	return m.name
}

// TemplateId returns the pet's template identifier
func (m Model) TemplateId() uint32 {
	return m.templateId
}

// Level returns the pet's level
func (m Model) Level() byte {
	return m.level
}

// Slot returns the pet's slot position
func (m Model) Slot() int8 {
	return m.slot
}

// IsSpawned returns true if the pet is currently spawned (slot >= 0)
func (m Model) IsSpawned() bool {
	return m.slot >= 0
}
