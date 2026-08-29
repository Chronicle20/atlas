package mock

import (
	"atlas-maker/data/itemmake"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

type ProcessorMock struct {
	GetAllFunc  func() ([]itemmake.Model, error)
	GetByIdFunc func(itemId item.Id) (itemmake.Model, error)
}

var _ itemmake.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetAll() ([]itemmake.Model, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return nil, nil
}

func (m *ProcessorMock) GetById(itemId item.Id) (itemmake.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(itemId)
	}
	return itemmake.Model{}, nil
}
