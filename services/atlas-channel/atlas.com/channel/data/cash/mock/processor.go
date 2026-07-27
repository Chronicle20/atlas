package mock

import (
	"atlas-channel/data/cash"
)

type ProcessorMock struct {
	GetByIdFunc func(itemId uint32) (cash.RestModel, error)
}

var _ cash.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(itemId uint32) (cash.RestModel, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(itemId)
	}
	return cash.RestModel{}, nil
}
