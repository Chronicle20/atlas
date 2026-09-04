package mock

import (
	"atlas-channel/data/monster"
)

type ProcessorMock struct {
	GetByIdFunc func(monsterId uint32) (monster.Model, error)
}

var _ monster.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(monsterId uint32) (monster.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(monsterId)
	}
	return monster.Model{}, nil
}
