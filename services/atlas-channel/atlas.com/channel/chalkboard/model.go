package chalkboard

type Model struct {
	id      uint32
	message string
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Message() string {
	return m.message
}
