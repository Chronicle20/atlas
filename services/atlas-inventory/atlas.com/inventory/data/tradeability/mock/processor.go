package mock

import (
	"atlas-inventory/data/tradeability"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByIdProviderFunc func(inventoryType inventory.Type, templateId item.Id) model.Provider[tradeability.Model]
	GetFunc          func(inventoryType inventory.Type, templateId item.Id) (tradeability.Model, error)
}

var _ tradeability.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByIdProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[tradeability.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(inventoryType, templateId)
	}
	return model.FixedProvider(tradeability.Model{})
}

func (m *ProcessorMock) Get(inventoryType inventory.Type, templateId item.Id) (tradeability.Model, error) {
	if m.GetFunc != nil {
		return m.GetFunc(inventoryType, templateId)
	}
	return tradeability.Model{}, nil
}
