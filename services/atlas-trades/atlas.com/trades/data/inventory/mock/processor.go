package mock

import (
	inventorydata "atlas-trades/data/inventory"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ProcessorMock is the injectable double for the inventory REST client. Each
// Func field defaults to a zero-valued success when left unset.
type ProcessorMock struct {
	CompartmentProviderFunc func(characterId character.Id, inventoryType inventory.Type) model.Provider[inventorydata.Model]
	GetCompartmentFunc      func(characterId character.Id, inventoryType inventory.Type) (inventorydata.Model, error)
	AssetInSlotFunc         func(characterId character.Id, inventoryType inventory.Type, s slot.Position) (inventorydata.Asset, error)
}

func (m *ProcessorMock) CompartmentProvider(characterId character.Id, inventoryType inventory.Type) model.Provider[inventorydata.Model] {
	if m.CompartmentProviderFunc != nil {
		return m.CompartmentProviderFunc(characterId, inventoryType)
	}
	return model.FixedProvider(inventorydata.Model{})
}

func (m *ProcessorMock) GetCompartment(characterId character.Id, inventoryType inventory.Type) (inventorydata.Model, error) {
	if m.GetCompartmentFunc != nil {
		return m.GetCompartmentFunc(characterId, inventoryType)
	}
	return inventorydata.Model{}, nil
}

func (m *ProcessorMock) AssetInSlot(characterId character.Id, inventoryType inventory.Type, s slot.Position) (inventorydata.Asset, error) {
	if m.AssetInSlotFunc != nil {
		return m.AssetInSlotFunc(characterId, inventoryType, s)
	}
	return inventorydata.Asset{}, nil
}

var _ inventorydata.Processor = (*ProcessorMock)(nil)
