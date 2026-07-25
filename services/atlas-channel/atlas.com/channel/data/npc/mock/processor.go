package mock

import (
	"atlas-channel/data/npc"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ForEachInMapFunc                 func(mapId _map.Id, f model.Operator[npc.Model]) error
	InMapModelProviderFunc           func(mapId _map.Id) model.Provider[[]npc.Model]
	InMapByObjectIdModelProviderFunc func(mapId _map.Id, objectId uint32) model.Provider[[]npc.Model]
	GetInMapByObjectIdFunc           func(mapId _map.Id, objectId uint32) (npc.Model, error)
}

var _ npc.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ForEachInMap(mapId _map.Id, f model.Operator[npc.Model]) error {
	if m.ForEachInMapFunc != nil {
		return m.ForEachInMapFunc(mapId, f)
	}
	return nil
}

func (m *ProcessorMock) InMapModelProvider(mapId _map.Id) model.Provider[[]npc.Model] {
	if m.InMapModelProviderFunc != nil {
		return m.InMapModelProviderFunc(mapId)
	}
	return model.FixedProvider([]npc.Model{})
}

func (m *ProcessorMock) InMapByObjectIdModelProvider(mapId _map.Id, objectId uint32) model.Provider[[]npc.Model] {
	if m.InMapByObjectIdModelProviderFunc != nil {
		return m.InMapByObjectIdModelProviderFunc(mapId, objectId)
	}
	return model.FixedProvider([]npc.Model{})
}

func (m *ProcessorMock) GetInMapByObjectId(mapId _map.Id, objectId uint32) (npc.Model, error) {
	if m.GetInMapByObjectIdFunc != nil {
		return m.GetInMapByObjectIdFunc(mapId, objectId)
	}
	return npc.Model{}, nil
}
