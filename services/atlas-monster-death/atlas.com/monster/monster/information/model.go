package information

type Model struct {
	hp         uint32
	experience uint32
	level      uint32
	name       string
}

func (m Model) Hp() uint32 {
	return m.hp
}

func (m Model) Experience() uint32 {
	return m.experience
}

func (m Model) Level() uint32 {
	return m.level
}

func (m Model) Name() string {
	return m.name
}
