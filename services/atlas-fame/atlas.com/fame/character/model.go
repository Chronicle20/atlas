package character

type Model struct {
	id    uint32
	name  string
	level byte
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Level() byte {
	return m.level
}
