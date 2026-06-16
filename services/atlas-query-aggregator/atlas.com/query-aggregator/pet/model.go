package pet

// Model represents a pet in the query aggregator
type Model struct {
	id         uint32
	slot       int8
	templateId uint32
	closeness  uint16
}

func NewModel(id uint32, slot int8, templateId uint32, closeness uint16) Model {
	return Model{
		id:         id,
		slot:       slot,
		templateId: templateId,
		closeness:  closeness,
	}
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Slot() int8 {
	return m.slot
}

func (m Model) TemplateId() uint32 {
	return m.templateId
}

func (m Model) Closeness() uint16 {
	return m.closeness
}

// IsSpawned returns true if the pet is currently spawned (slot >= 0)
func (m Model) IsSpawned() bool {
	return m.slot >= 0
}
