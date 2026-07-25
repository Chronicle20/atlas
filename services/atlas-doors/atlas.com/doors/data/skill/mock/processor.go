package mock

import (
	skilldata "atlas-doors/data/skill"
	"atlas-doors/data/skill/effect"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

type ProcessorMock struct {
	GetByIdFunc   func(skillId skill.Id) (skilldata.Model, error)
	GetEffectFunc func(skillId skill.Id, level byte) (effect.Model, error)
}

var _ skilldata.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(skillId skill.Id) (skilldata.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(skillId)
	}
	return skilldata.Model{}, nil
}

func (m *ProcessorMock) GetEffect(skillId skill.Id, level byte) (effect.Model, error) {
	if m.GetEffectFunc != nil {
		return m.GetEffectFunc(skillId, level)
	}
	return effect.Model{}, nil
}
