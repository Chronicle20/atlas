package macro

import "github.com/Chronicle20/atlas-constants/skill"

type Model struct {
	id       uint32
	name     string
	shout    bool
	skillId1 skill.Id
	skillId2 skill.Id
	skillId3 skill.Id
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Shout() bool {
	return m.shout
}

func (m Model) SkillId1() skill.Id {
	return m.skillId1
}

func (m Model) SkillId2() skill.Id {
	return m.skillId2
}

func (m Model) SkillId3() skill.Id {
	return m.skillId3
}
