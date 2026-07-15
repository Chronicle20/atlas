package mock

import (
	"atlas-storage/asset"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	GetAssetByIdFunc         func(assetId uint32) (asset.Model, error)
	GetAssetsByStorageIdFunc func(storageId uuid.UUID) ([]asset.Model, error)
	GetOrCreateStorageIdFunc func(worldId world.Id, accountId uint32) (uuid.UUID, error)
}

var _ asset.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetAssetById(assetId uint32) (asset.Model, error) {
	if m.GetAssetByIdFunc != nil {
		return m.GetAssetByIdFunc(assetId)
	}
	return asset.Model{}, nil
}

func (m *ProcessorMock) GetAssetsByStorageId(storageId uuid.UUID) ([]asset.Model, error) {
	if m.GetAssetsByStorageIdFunc != nil {
		return m.GetAssetsByStorageIdFunc(storageId)
	}
	return nil, nil
}

func (m *ProcessorMock) GetOrCreateStorageId(worldId world.Id, accountId uint32) (uuid.UUID, error) {
	if m.GetOrCreateStorageIdFunc != nil {
		return m.GetOrCreateStorageIdFunc(worldId, accountId)
	}
	return uuid.Nil, nil
}
