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
	SlotMaxProviderFunc    func(inventoryType inventory.Type, templateId item.Id) model.Provider[uint32]
	SlotMaxFunc            func(inventoryType inventory.Type, templateId item.Id) (uint32, error)
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

func (m *ProcessorMock) SlotMaxProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[uint32] {
	if m.SlotMaxProviderFunc != nil {
		return m.SlotMaxProviderFunc(inventoryType, templateId)
	}
	return model.FixedProvider(uint32(1))
}

func (m *ProcessorMock) SlotMax(inventoryType inventory.Type, templateId item.Id) (uint32, error) {
	if m.SlotMaxFunc != nil {
		return m.SlotMaxFunc(inventoryType, templateId)
	}
	return 1, nil
}

var _ itemdata.Processor = (*ProcessorMock)(nil)
