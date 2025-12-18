package title

type Model struct {
	name  string
	index byte
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Index() byte {
	return m.index
}
