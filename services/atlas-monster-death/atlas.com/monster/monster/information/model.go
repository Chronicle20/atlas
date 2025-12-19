package information

type Model struct {
	hp         uint32
	experience uint32
}

func (m Model) HP() uint32 {
	return m.hp
}

func (m Model) Experience() uint32 {
	return m.experience
}
