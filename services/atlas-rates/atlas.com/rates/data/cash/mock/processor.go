package mock

import (
	"atlas-rates/data/cash"
)

type ProcessorMock struct {
	GetByIdFunc func(id uint32) (cash.RestModel, error)
}

var _ cash.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(id uint32) (cash.RestModel, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return cash.RestModel{}, nil
}
