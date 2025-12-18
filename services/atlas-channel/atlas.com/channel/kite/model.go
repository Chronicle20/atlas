package kite

type Model struct {
	id         uint32
	templateId uint32
	message    string
	name       string
	x          int16
	y          int16
	ft         int16
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) TemplateId() uint32 {
	return m.templateId
}

func (m Model) Message() string {
	return m.message
}

func (m Model) Name() string {
	return m.name
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m Model) Type() int16 {
	return m.ft
}
