package cashpackage

type Model struct {
	id            uint32
	serialNumbers []uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) SerialNumbers() []uint32 {
	return m.serialNumbers
}
