package mock

import (
	"atlas-channel/data/skill"
	"atlas-channel/data/skill/effect"
)

type ProcessorMock struct {
	GetByIdFunc   func(uniqueId uint32) (skill.Model, error)
	GetEffectFunc func(uniqueId uint32, level byte) (effect.Model, error)
}

var _ skill.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(uniqueId uint32) (skill.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(uniqueId)
	}
	return skill.Model{}, nil
}

func (m *ProcessorMock) GetEffect(uniqueId uint32, level byte) (effect.Model, error) {
	if m.GetEffectFunc != nil {
		return m.GetEffectFunc(uniqueId, level)
	}
	return effect.Model{}, nil
}
