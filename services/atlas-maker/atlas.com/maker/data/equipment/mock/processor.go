package mock

import (
	"atlas-maker/data/equipment"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

type ProcessorMock struct {
	GetByIdFunc func(itemId item.Id) (equipment.Model, error)
}

var _ equipment.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(itemId item.Id) (equipment.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(itemId)
	}
	return equipment.Model{}, nil
}
