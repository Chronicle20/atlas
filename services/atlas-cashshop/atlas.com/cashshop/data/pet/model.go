package pet

type Model struct {
	id   uint32
	name string
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Name() string {
	return m.name
}
