package monster

type Model struct {
	id          uint32
	boss        bool
	fixedDamage uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Boss() bool {
	return m.boss
}

func (m Model) FixedDamage() uint32 {
	return m.fixedDamage
}
