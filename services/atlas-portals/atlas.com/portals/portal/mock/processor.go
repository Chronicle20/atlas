package mock

import (
	"atlas-portals/portal"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	InMapByNameProviderFunc func(mapId _map.Id, name string) model.Provider[[]portal.Model]
	InMapByIdProviderFunc   func(mapId _map.Id, id uint32) model.Provider[portal.Model]
	GetInMapByNameFunc      func(mapId _map.Id, name string) (portal.Model, error)
	GetInMapByIdFunc        func(mapId _map.Id, id uint32) (portal.Model, error)
	InMapProviderFunc       func(mapId _map.Id) model.Provider[[]portal.Model]
	WarpFunc                func(f field.Model, characterId uint32, targetMapId _map.Id)
	EnterFunc               func(f field.Model, portalId uint32, characterId uint32)
	WarpByIdFunc            func(f field.Model, characterId uint32, targetMapId _map.Id, portalId uint32)
	WarpToPositionFunc      func(f field.Model, characterId uint32, targetMapId _map.Id, x int16, y int16)
	WarpToPortalFunc        func(f field.Model, characterId uint32, targetMapId _map.Id, portalProvider model.Provider[uint32])
}

var _ portal.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) InMapByNameProvider(mapId _map.Id, name string) model.Provider[[]portal.Model] {
	if m.InMapByNameProviderFunc != nil {
		return m.InMapByNameProviderFunc(mapId, name)
	}
	return model.FixedProvider([]portal.Model{})
}

func (m *ProcessorMock) InMapByIdProvider(mapId _map.Id, id uint32) model.Provider[portal.Model] {
	if m.InMapByIdProviderFunc != nil {
		return m.InMapByIdProviderFunc(mapId, id)
	}
	return model.FixedProvider(portal.Model{})
}

func (m *ProcessorMock) GetInMapByName(mapId _map.Id, name string) (portal.Model, error) {
	if m.GetInMapByNameFunc != nil {
		return m.GetInMapByNameFunc(mapId, name)
	}
	return portal.Model{}, nil
}

func (m *ProcessorMock) GetInMapById(mapId _map.Id, id uint32) (portal.Model, error) {
	if m.GetInMapByIdFunc != nil {
		return m.GetInMapByIdFunc(mapId, id)
	}
	return portal.Model{}, nil
}

func (m *ProcessorMock) InMapProvider(mapId _map.Id) model.Provider[[]portal.Model] {
	if m.InMapProviderFunc != nil {
		return m.InMapProviderFunc(mapId)
	}
	return model.FixedProvider([]portal.Model{})
}

func (m *ProcessorMock) Warp(f field.Model, characterId uint32, targetMapId _map.Id) {
	if m.WarpFunc != nil {
		m.WarpFunc(f, characterId, targetMapId)
	}
}

func (m *ProcessorMock) Enter(f field.Model, portalId uint32, characterId uint32) {
	if m.EnterFunc != nil {
		m.EnterFunc(f, portalId, characterId)
	}
}

func (m *ProcessorMock) WarpById(f field.Model, characterId uint32, targetMapId _map.Id, portalId uint32) {
	if m.WarpByIdFunc != nil {
		m.WarpByIdFunc(f, characterId, targetMapId, portalId)
	}
}

func (m *ProcessorMock) WarpToPosition(f field.Model, characterId uint32, targetMapId _map.Id, x int16, y int16) {
	if m.WarpToPositionFunc != nil {
		m.WarpToPositionFunc(f, characterId, targetMapId, x, y)
	}
}

func (m *ProcessorMock) WarpToPortal(f field.Model, characterId uint32, targetMapId _map.Id, portalProvider model.Provider[uint32]) {
	if m.WarpToPortalFunc != nil {
		m.WarpToPortalFunc(f, characterId, targetMapId, portalProvider)
	}
}
