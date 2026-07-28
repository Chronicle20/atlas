package mock

import (
	"atlas-consumables/data/consumable"
)

type ProcessorMock struct {
	GetByIdFunc func(itemId uint32) (consumable.Model, error)
}

var _ consumable.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(itemId uint32) (consumable.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(itemId)
	}
	return consumable.Model{}, nil
}
