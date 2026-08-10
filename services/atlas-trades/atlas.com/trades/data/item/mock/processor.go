package mock

import (
	itemdata "atlas-trades/data/item"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ProcessorMock is the injectable double for the atlas-data item reader. Each
// Func field defaults to a zero-valued success when left unset.
type ProcessorMock struct {
	TradeBlockProviderFunc func(inventoryType inventory.Type, templateId item.Id) model.Provider[bool]
	TradeBlockFunc         func(inventoryType inventory.Type, templateId item.Id) (bool, error)
}

func (m *ProcessorMock) TradeBlockProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[bool] {
	if m.TradeBlockProviderFunc != nil {
		return m.TradeBlockProviderFunc(inventoryType, templateId)
	}
	return model.FixedProvider(false)
}

func (m *ProcessorMock) TradeBlock(inventoryType inventory.Type, templateId item.Id) (bool, error) {
	if m.TradeBlockFunc != nil {
		return m.TradeBlockFunc(inventoryType, templateId)
	}
	return false, nil
}

var _ itemdata.Processor = (*ProcessorMock)(nil)
