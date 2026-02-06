package mock

import (
	"atlas-saga-orchestrator/data/portal"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorMock is a mock implementation of the portal.Processor interface
type ProcessorMock struct {
	InMapProviderFunc              func(mapId _map.Id) model.Provider[[]portal.Model]
	RandomSpawnPointProviderFunc   func(mapId _map.Id) model.Provider[portal.Model]
	RandomSpawnPointIdProviderFunc func(mapId _map.Id) model.Provider[uint32]
	ByNameIdProviderFunc           func(mapId _map.Id, name string) model.Provider[uint32]
}

// InMapProvider is a mock implementation of the portal.Processor.InMapProvider method
func (m *ProcessorMock) InMapProvider(mapId _map.Id) model.Provider[[]portal.Model] {
	if m.InMapProviderFunc != nil {
		return m.InMapProviderFunc(mapId)
	}
	return func() ([]portal.Model, error) {
		return nil, nil
	}
}

// RandomSpawnPointProvider is a mock implementation of the portal.Processor.RandomSpawnPointProvider method
func (m *ProcessorMock) RandomSpawnPointProvider(mapId _map.Id) model.Provider[portal.Model] {
	if m.RandomSpawnPointProviderFunc != nil {
		return m.RandomSpawnPointProviderFunc(mapId)
	}
	return func() (portal.Model, error) {
		return portal.Model{}, nil
	}
}

// RandomSpawnPointIdProvider is a mock implementation of the portal.Processor.RandomSpawnPointIdProvider method
func (m *ProcessorMock) RandomSpawnPointIdProvider(mapId _map.Id) model.Provider[uint32] {
	if m.RandomSpawnPointIdProviderFunc != nil {
		return m.RandomSpawnPointIdProviderFunc(mapId)
	}
	return func() (uint32, error) {
		return 0, nil
	}
}

// ByNameIdProvider is a mock implementation of the portal.Processor.ByNameIdProvider method
func (m *ProcessorMock) ByNameIdProvider(mapId _map.Id, name string) model.Provider[uint32] {
	if m.ByNameIdProviderFunc != nil {
		return m.ByNameIdProviderFunc(mapId, name)
	}
	return func() (uint32, error) {
		return 0, nil
	}
}
