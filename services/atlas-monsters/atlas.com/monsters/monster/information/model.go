package information

type Model struct {
	hp uint32
	mp uint32
}

func (m Model) HP() uint32 {
	return m.hp
}

func (m Model) MP() uint32 {
	return m.mp
}
