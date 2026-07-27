package mock

import (
	"atlas-rates/data/equipment"
)

type ProcessorMock struct {
	GetByIdFunc func(id uint32) (equipment.RestModel, error)
}

var _ equipment.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(id uint32) (equipment.RestModel, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return equipment.RestModel{}, nil
}
