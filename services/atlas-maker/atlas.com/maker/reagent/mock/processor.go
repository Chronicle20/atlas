package mock

import (
	"atlas-maker/reagent"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	GetAllFunc      func() ([]reagent.Model, error)
	GetAllPagedFunc func(page model.Page) model.Provider[model.Paged[reagent.Model]]
	GetByItemIdFunc func(itemId item.Id) (reagent.Model, error)
}

var _ reagent.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetAll() ([]reagent.Model, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return nil, nil
}

func (m *ProcessorMock) GetAllPaged(page model.Page) model.Provider[model.Paged[reagent.Model]] {
	if m.GetAllPagedFunc != nil {
		return m.GetAllPagedFunc(page)
	}
	return model.FixedProvider(model.Paged[reagent.Model]{})
}

func (m *ProcessorMock) GetByItemId(itemId item.Id) (reagent.Model, error) {
	if m.GetByItemIdFunc != nil {
		return m.GetByItemIdFunc(itemId)
	}
	return reagent.Model{}, nil
}
