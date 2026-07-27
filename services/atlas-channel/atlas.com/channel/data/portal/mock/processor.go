package mock

import (
	"atlas-channel/data/portal"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	InMapByNameModelProviderFunc func(mapId _map.Id, name string) model.Provider[[]portal.Model]
	GetInMapByNameFunc           func(mapId _map.Id, name string) (portal.Model, error)
}

var _ portal.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) InMapByNameModelProvider(mapId _map.Id, name string) model.Provider[[]portal.Model] {
	if m.InMapByNameModelProviderFunc != nil {
		return m.InMapByNameModelProviderFunc(mapId, name)
	}
	return model.FixedProvider([]portal.Model{})
}

func (m *ProcessorMock) GetInMapByName(mapId _map.Id, name string) (portal.Model, error) {
	if m.GetInMapByNameFunc != nil {
		return m.GetInMapByNameFunc(mapId, name)
	}
	return portal.Model{}, nil
}
