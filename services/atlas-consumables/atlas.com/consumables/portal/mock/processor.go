package mock

import (
	"atlas-consumables/portal"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	InMapProviderFunc              func(mapId _map.Id) model.Provider[[]portal.Model]
	RandomSpawnPointProviderFunc   func(mapId _map.Id) model.Provider[portal.Model]
	RandomSpawnPointIdProviderFunc func(mapId _map.Id) model.Provider[uint32]
}

var _ portal.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) InMapProvider(mapId _map.Id) model.Provider[[]portal.Model] {
	if m.InMapProviderFunc != nil {
		return m.InMapProviderFunc(mapId)
	}
	return model.FixedProvider([]portal.Model{})
}

func (m *ProcessorMock) RandomSpawnPointProvider(mapId _map.Id) model.Provider[portal.Model] {
	if m.RandomSpawnPointProviderFunc != nil {
		return m.RandomSpawnPointProviderFunc(mapId)
	}
	return model.FixedProvider(portal.Model{})
}

func (m *ProcessorMock) RandomSpawnPointIdProvider(mapId _map.Id) model.Provider[uint32] {
	if m.RandomSpawnPointIdProviderFunc != nil {
		return m.RandomSpawnPointIdProviderFunc(mapId)
	}
	return model.FixedProvider(uint32(0))
}
