package mock

import (
	"atlas-asset-expiration/storage"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type ProcessorMock struct {
	GetAssetsFunc func(accountId uint32, worldId world.Id) ([]storage.AssetRestModel, error)
}

var _ storage.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetAssets(accountId uint32, worldId world.Id) ([]storage.AssetRestModel, error) {
	if m.GetAssetsFunc != nil {
		return m.GetAssetsFunc(accountId, worldId)
	}
	return nil, nil
}
