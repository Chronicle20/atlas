package mock

import (
	"atlas-monsters/monster/mobskill"
)

type ProcessorMock struct {
	GetByIdAndLevelFunc func(skillId uint16, level uint16) (mobskill.Model, error)
}

var _ mobskill.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetByIdAndLevel(skillId uint16, level uint16) (mobskill.Model, error) {
	if m.GetByIdAndLevelFunc != nil {
		return m.GetByIdAndLevelFunc(skillId, level)
	}
	return mobskill.Model{}, nil
}
