package character

type Model struct {
	id    uint32
	level byte
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Level() byte {
	return m.level
}
