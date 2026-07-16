package mock

import (
	"atlas-asset-expiration/inventory"
)

type ProcessorMock struct {
	GetInventoryFunc func(characterId uint32) (inventory.RestModel, error)
	GetAssetsFunc    func(characterId uint32, compartmentId string) ([]inventory.AssetRestModel, error)
}

var _ inventory.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetInventory(characterId uint32) (inventory.RestModel, error) {
	if m.GetInventoryFunc != nil {
		return m.GetInventoryFunc(characterId)
	}
	return inventory.RestModel{}, nil
}

func (m *ProcessorMock) GetAssets(characterId uint32, compartmentId string) ([]inventory.AssetRestModel, error) {
	if m.GetAssetsFunc != nil {
		return m.GetAssetsFunc(characterId, compartmentId)
	}
	return nil, nil
}
