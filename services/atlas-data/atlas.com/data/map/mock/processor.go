package mock

import (
	_map "atlas-data/map"
	"atlas-data/map/monster"
	"atlas-data/map/npc"
	"atlas-data/map/portal"
	"atlas-data/map/reactor"
	monstertpl "atlas-data/monster"

	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc          func(s *_map.Storage, r model.Provider[_map.RestModel]) error
	RegisterMapFunc       func(path string) error
	GetPortalsFunc        func(s *_map.Storage, mapId mapconst.Id) ([]portal.RestModel, error)
	GetPortalsByNameFunc  func(s *_map.Storage, mapId mapconst.Id, name string) ([]portal.RestModel, error)
	GetPortalByIdFunc     func(s *_map.Storage, mapId mapconst.Id, portalId uint32) (portal.RestModel, error)
	GetReactorsFunc       func(s *_map.Storage, mapId mapconst.Id) ([]reactor.RestModel, error)
	GetNpcsFunc           func(s *_map.Storage, mapId mapconst.Id) ([]npc.RestModel, error)
	GetNpcsByObjectIdFunc func(s *_map.Storage, mapId mapconst.Id, objectId uint32) ([]npc.RestModel, error)
	GetNpcFunc            func(s *_map.Storage, mapId mapconst.Id, npcId uint32) (npc.RestModel, error)
	GetMonstersFunc       func(s *_map.Storage, ms *monstertpl.Storage, mapId mapconst.Id) ([]monster.RestModel, error)
}

var _ _map.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *_map.Storage, r model.Provider[_map.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterMap(path string) error {
	if m.RegisterMapFunc != nil {
		return m.RegisterMapFunc(path)
	}
	return nil
}

func (m *ProcessorMock) GetPortals(s *_map.Storage, mapId mapconst.Id) ([]portal.RestModel, error) {
	if m.GetPortalsFunc != nil {
		return m.GetPortalsFunc(s, mapId)
	}
	return nil, nil
}

func (m *ProcessorMock) GetPortalsByName(s *_map.Storage, mapId mapconst.Id, name string) ([]portal.RestModel, error) {
	if m.GetPortalsByNameFunc != nil {
		return m.GetPortalsByNameFunc(s, mapId, name)
	}
	return nil, nil
}

func (m *ProcessorMock) GetPortalById(s *_map.Storage, mapId mapconst.Id, portalId uint32) (portal.RestModel, error) {
	if m.GetPortalByIdFunc != nil {
		return m.GetPortalByIdFunc(s, mapId, portalId)
	}
	return portal.RestModel{}, nil
}

func (m *ProcessorMock) GetReactors(s *_map.Storage, mapId mapconst.Id) ([]reactor.RestModel, error) {
	if m.GetReactorsFunc != nil {
		return m.GetReactorsFunc(s, mapId)
	}
	return nil, nil
}

func (m *ProcessorMock) GetNpcs(s *_map.Storage, mapId mapconst.Id) ([]npc.RestModel, error) {
	if m.GetNpcsFunc != nil {
		return m.GetNpcsFunc(s, mapId)
	}
	return nil, nil
}

func (m *ProcessorMock) GetNpcsByObjectId(s *_map.Storage, mapId mapconst.Id, objectId uint32) ([]npc.RestModel, error) {
	if m.GetNpcsByObjectIdFunc != nil {
		return m.GetNpcsByObjectIdFunc(s, mapId, objectId)
	}
	return nil, nil
}

func (m *ProcessorMock) GetNpc(s *_map.Storage, mapId mapconst.Id, npcId uint32) (npc.RestModel, error) {
	if m.GetNpcFunc != nil {
		return m.GetNpcFunc(s, mapId, npcId)
	}
	return npc.RestModel{}, nil
}

func (m *ProcessorMock) GetMonsters(s *_map.Storage, ms *monstertpl.Storage, mapId mapconst.Id) ([]monster.RestModel, error) {
	if m.GetMonstersFunc != nil {
		return m.GetMonstersFunc(s, ms, mapId)
	}
	return nil, nil
}
