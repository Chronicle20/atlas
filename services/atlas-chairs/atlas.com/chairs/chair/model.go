package chair

type Model struct {
	id        uint32
	chairType string
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Type() string {
	return m.chairType
}
