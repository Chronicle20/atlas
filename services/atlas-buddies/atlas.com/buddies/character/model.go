package character

type Model struct {
	id   uint32
	name string
	gm   int
}

func (m Model) Name() string {
	return m.name
}

func (m Model) GM() int {
	return m.gm
}
