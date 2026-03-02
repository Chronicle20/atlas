package pet

type Model struct {
	id         uint32
	templateId uint32
	name       string
	ownerId    uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) TemplateId() uint32 {
	return m.templateId
}

func (m Model) Name() string {
	return m.name
}

func (m Model) OwnerId() uint32 {
	return m.ownerId
}
