package macro

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

type builder struct {
	id       uint32
	name     string
	shout    bool
	skillId1 skill.Id
	skillId2 skill.Id
	skillId3 skill.Id
}

func NewBuilder() *builder {
	return &builder{}
}

func CloneModel(m Model) *builder {
	return &builder{
		id:       m.id,
		name:     m.name,
		shout:    m.shout,
		skillId1: m.skillId1,
		skillId2: m.skillId2,
		skillId3: m.skillId3,
	}
}

func (b *builder) SetId(id uint32) *builder {
	b.id = id
	return b
}

func (b *builder) SetName(name string) *builder {
	b.name = name
	return b
}

func (b *builder) SetShout(shout bool) *builder {
	b.shout = shout
	return b
}

func (b *builder) SetSkillId1(skillId skill.Id) *builder {
	b.skillId1 = skillId
	return b
}

func (b *builder) SetSkillId2(skillId skill.Id) *builder {
	b.skillId2 = skillId
	return b
}

func (b *builder) SetSkillId3(skillId skill.Id) *builder {
	b.skillId3 = skillId
	return b
}

func (b *builder) Build() (Model, error) {
	return Model{
		id:       b.id,
		name:     b.name,
		shout:    b.shout,
		skillId1: b.skillId1,
		skillId2: b.skillId2,
		skillId3: b.skillId3,
	}, nil
}
