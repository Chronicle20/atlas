package macro

import (
	"errors"

	"github.com/Chronicle20/atlas-constants/skill"
)

type modelBuilder struct {
	id       uint32
	name     string
	shout    bool
	skillId1 skill.Id
	skillId2 skill.Id
	skillId3 skill.Id
}

func NewModelBuilder() *modelBuilder {
	return &modelBuilder{}
}

func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:       m.id,
		name:     m.name,
		shout:    m.shout,
		skillId1: m.skillId1,
		skillId2: m.skillId2,
		skillId3: m.skillId3,
	}
}

func (b *modelBuilder) SetId(id uint32) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetName(name string) *modelBuilder {
	b.name = name
	return b
}

func (b *modelBuilder) SetShout(shout bool) *modelBuilder {
	b.shout = shout
	return b
}

func (b *modelBuilder) SetSkillId1(skillId skill.Id) *modelBuilder {
	b.skillId1 = skillId
	return b
}

func (b *modelBuilder) SetSkillId2(skillId skill.Id) *modelBuilder {
	b.skillId2 = skillId
	return b
}

func (b *modelBuilder) SetSkillId3(skillId skill.Id) *modelBuilder {
	b.skillId3 = skillId
	return b
}

func (b *modelBuilder) Build() (Model, error) {
	if b.name == "" {
		return Model{}, ErrMissingName
	}
	return Model{
		id:       b.id,
		name:     b.name,
		shout:    b.shout,
		skillId1: b.skillId1,
		skillId2: b.skillId2,
		skillId3: b.skillId3,
	}, nil
}

var (
	ErrMissingName = errors.New("macro name is required")
)
