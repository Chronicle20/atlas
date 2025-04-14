package asset

type Model struct {
	id         uint32
	slot       int16
	templateId uint32
}

func (m Model) Slot() int16 {
	return m.slot
}

func (m Model) Id() uint32 {
	return m.id
}

func NewModel(id uint32, templateId uint32, slot int16) Model {
	return Model{
		id:         id,
		slot:       slot,
		templateId: templateId,
	}
}
