package mock

import (
	_map "atlas-channel/map"
	"atlas-channel/session"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	CharacterIdsInMapModelProviderFunc             func(f field.Model) model.Provider[[]uint32]
	GetCharacterIdsInMapFunc                       func(f field.Model) ([]uint32, error)
	ForSessionsInSessionsMapFunc                   func(f func(oid uint32) model.Operator[session.Model]) model.Operator[session.Model]
	ForSessionsInMapFunc                           func(f field.Model, o model.Operator[session.Model]) error
	CharacterIdsInMapAllInstancesModelProviderFunc func(worldId world.Id, channelId channel.Id, mapId mapconst.Id) model.Provider[[]uint32]
	ForSessionsInMapAllInstancesFunc               func(worldId world.Id, channelId channel.Id, mapId mapconst.Id, o model.Operator[session.Model]) error
	OtherCharacterIdsInMapModelProviderFunc        func(f field.Model, referenceCharacterId uint32) model.Provider[[]uint32]
	ForOtherSessionsInMapFunc                      func(f field.Model, referenceCharacterId uint32, o model.Operator[session.Model]) error
}

var _ _map.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) CharacterIdsInMapModelProvider(f field.Model) model.Provider[[]uint32] {
	if m.CharacterIdsInMapModelProviderFunc != nil {
		return m.CharacterIdsInMapModelProviderFunc(f)
	}
	return model.FixedProvider([]uint32{})
}

func (m *ProcessorMock) GetCharacterIdsInMap(f field.Model) ([]uint32, error) {
	if m.GetCharacterIdsInMapFunc != nil {
		return m.GetCharacterIdsInMapFunc(f)
	}
	return nil, nil
}

func (m *ProcessorMock) ForSessionsInSessionsMap(f func(oid uint32) model.Operator[session.Model]) model.Operator[session.Model] {
	if m.ForSessionsInSessionsMapFunc != nil {
		return m.ForSessionsInSessionsMapFunc(f)
	}
	return func(s session.Model) error {
		return nil
	}
}

func (m *ProcessorMock) ForSessionsInMap(f field.Model, o model.Operator[session.Model]) error {
	if m.ForSessionsInMapFunc != nil {
		return m.ForSessionsInMapFunc(f, o)
	}
	return nil
}

func (m *ProcessorMock) CharacterIdsInMapAllInstancesModelProvider(worldId world.Id, channelId channel.Id, mapId mapconst.Id) model.Provider[[]uint32] {
	if m.CharacterIdsInMapAllInstancesModelProviderFunc != nil {
		return m.CharacterIdsInMapAllInstancesModelProviderFunc(worldId, channelId, mapId)
	}
	return model.FixedProvider([]uint32{})
}

func (m *ProcessorMock) ForSessionsInMapAllInstances(worldId world.Id, channelId channel.Id, mapId mapconst.Id, o model.Operator[session.Model]) error {
	if m.ForSessionsInMapAllInstancesFunc != nil {
		return m.ForSessionsInMapAllInstancesFunc(worldId, channelId, mapId, o)
	}
	return nil
}

func (m *ProcessorMock) OtherCharacterIdsInMapModelProvider(f field.Model, referenceCharacterId uint32) model.Provider[[]uint32] {
	if m.OtherCharacterIdsInMapModelProviderFunc != nil {
		return m.OtherCharacterIdsInMapModelProviderFunc(f, referenceCharacterId)
	}
	return model.FixedProvider([]uint32{})
}

func (m *ProcessorMock) ForOtherSessionsInMap(f field.Model, referenceCharacterId uint32, o model.Operator[session.Model]) error {
	if m.ForOtherSessionsInMapFunc != nil {
		return m.ForOtherSessionsInMapFunc(f, referenceCharacterId, o)
	}
	return nil
}
