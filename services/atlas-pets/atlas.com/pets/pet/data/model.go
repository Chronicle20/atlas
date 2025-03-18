package data

type Model struct {
	id     uint32
	hunger uint32
	cash   bool
	life   uint32
	skills []SkillModel
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Hunger() uint32 {
	return m.hunger
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

type SkillModel struct {
	id          string
	increase    uint16
	probability uint16
}

func (m SkillModel) Id() string {
	return m.id
}

func (m SkillModel) Probability() uint16 {
	return m.probability
}

func (m SkillModel) Increase() uint16 {
	return m.increase
}
