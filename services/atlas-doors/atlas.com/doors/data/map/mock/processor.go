package mock

import (
	mapdata "atlas-doors/data/map"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type ProcessorMock struct {
	GetByIdFunc    func(mapId _map.Id) (mapdata.Model, error)
	GetPortalsFunc func(mapId _map.Id) ([]mapdata.Portal, error)
}

var _ mapdata.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(mapId _map.Id) (mapdata.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(mapId)
	}
	return mapdata.Model{}, nil
}

func (m *ProcessorMock) GetPortals(mapId _map.Id) ([]mapdata.Portal, error) {
	if m.GetPortalsFunc != nil {
		return m.GetPortalsFunc(mapId)
	}
	return []mapdata.Portal{}, nil
}
