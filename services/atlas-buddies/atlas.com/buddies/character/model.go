package character

type Model struct {
	id   uint32
	name string
}

func (m Model) Name() string {
	return m.name
}
