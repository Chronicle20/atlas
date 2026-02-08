package information

type Model struct {
	hp uint32
	mp uint32
}

func (m Model) Hp() uint32 {
	return m.hp
}

func (m Model) Mp() uint32 {
	return m.mp
}
