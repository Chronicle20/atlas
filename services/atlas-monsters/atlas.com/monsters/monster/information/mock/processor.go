package mock

import (
	"atlas-monsters/monster/information"
)

type ProcessorMock struct {
	GetByIdFunc func(monsterId uint32) (information.Model, error)
}

var _ information.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(monsterId uint32) (information.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(monsterId)
	}
	return information.Model{}, nil
}
