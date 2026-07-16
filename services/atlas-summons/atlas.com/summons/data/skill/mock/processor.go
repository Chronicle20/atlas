package mock

import (
	"atlas-summons/data/skill"
	"atlas-summons/data/skill/effect"
)

type ProcessorMock struct {
	GetByIdFunc   func(skillId uint32) (skill.Model, error)
	GetEffectFunc func(skillId uint32, level byte) (effect.Model, error)
}

var _ skill.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(skillId uint32) (skill.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(skillId)
	}
	return skill.Model{}, nil
}

func (m *ProcessorMock) GetEffect(skillId uint32, level byte) (effect.Model, error) {
	if m.GetEffectFunc != nil {
		return m.GetEffectFunc(skillId, level)
	}
	return effect.Model{}, nil
}
