package mock

import (
	"atlas-maker/compartment"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByCharacterIdAndTypeProviderFunc func(characterId uint32, inventoryType inventory.Type) model.Provider[compartment.Model]
	GetByTypeFunc                    func(characterId uint32, inventoryType inventory.Type) (compartment.Model, error)
	CanAccommodateFunc               func(characterId uint32, items []compartment.AccommodationItem) (bool, error)
}

var _ compartment.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByCharacterIdAndTypeProvider(characterId uint32, inventoryType inventory.Type) model.Provider[compartment.Model] {
	if m.ByCharacterIdAndTypeProviderFunc != nil {
		return m.ByCharacterIdAndTypeProviderFunc(characterId, inventoryType)
	}
	return model.FixedProvider(compartment.Model{})
}

func (m *ProcessorMock) GetByType(characterId uint32, inventoryType inventory.Type) (compartment.Model, error) {
	if m.GetByTypeFunc != nil {
		return m.GetByTypeFunc(characterId, inventoryType)
	}
	return compartment.Model{}, nil
}

func (m *ProcessorMock) CanAccommodate(characterId uint32, items []compartment.AccommodationItem) (bool, error) {
	if m.CanAccommodateFunc != nil {
		return m.CanAccommodateFunc(characterId, items)
	}
	return true, nil
}
