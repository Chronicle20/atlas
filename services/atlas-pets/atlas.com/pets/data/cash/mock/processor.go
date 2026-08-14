package mock

import (
	"atlas-pets/data/cash"
)

// ProcessorMock is a function-field stub of cash.Processor.
type ProcessorMock struct {
	GetByIdFunc func(itemId uint32) (cash.Model, error)
}

var _ cash.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(itemId uint32) (cash.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(itemId)
	}
	return cash.Model{}, nil
}
