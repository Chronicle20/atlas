package mock

import (
	mapdata "atlas-trades/data/map"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ProcessorMock is the injectable double for the map REST client. Each Func
// field defaults to a zero-valued success when left unset.
type ProcessorMock struct {
	GetByIdFunc      func(mapId _map.Id) (mapdata.Model, error)
	ByIdProviderFunc func(mapId _map.Id) model.Provider[mapdata.Model]
	FieldLimitFunc   func(mapId _map.Id) (uint32, error)
}

func (m *ProcessorMock) GetById(mapId _map.Id) (mapdata.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(mapId)
	}
	return mapdata.Model{}, nil
}

func (m *ProcessorMock) ByIdProvider(mapId _map.Id) model.Provider[mapdata.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(mapId)
	}
	return model.FixedProvider(mapdata.Model{})
}

func (m *ProcessorMock) FieldLimit(mapId _map.Id) (uint32, error) {
	if m.FieldLimitFunc != nil {
		return m.FieldLimitFunc(mapId)
	}
	return 0, nil
}

var _ mapdata.Processor = (*ProcessorMock)(nil)
