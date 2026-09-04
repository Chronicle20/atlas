package mock

import (
	"atlas-maker/recipe"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

type ProcessorMock struct {
	GetByIdFunc       func(itemId item.Id) (recipe.Model, error)
	GetByLeftoverFunc func(leftoverItemId item.Id) (recipe.Model, error)
	GetAllFunc        func() ([]recipe.Model, error)
}

var _ recipe.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(itemId item.Id) (recipe.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(itemId)
	}
	return recipe.Model{}, nil
}

func (m *ProcessorMock) GetByLeftover(leftoverItemId item.Id) (recipe.Model, error) {
	if m.GetByLeftoverFunc != nil {
		return m.GetByLeftoverFunc(leftoverItemId)
	}
	return recipe.Model{}, nil
}

func (m *ProcessorMock) GetAll() ([]recipe.Model, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return nil, nil
}
