package pet

type Model struct {
	id     uint64
	hungry uint32
	cash   bool
	life   uint32
	skills []SkillModel
}

type SkillModel struct {
	id          string
	increase    uint16
	probability uint16
}

func (m Model) Id() uint64 {
	return m.id
}

func (m Model) Hungry() uint32 {
	return m.hungry
}

func (m Model) Cash() bool {
	return m.cash
}

func (m Model) Life() uint32 {
	return m.life
}

func (m Model) Skills() []SkillModel {
	return m.skills
}
