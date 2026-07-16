package mock

import (
	"atlas-channel/door"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	InFieldModelProviderFunc func(f field.Model) model.Provider[[]door.Model]
	GetInFieldFunc           func(f field.Model) ([]door.Model, error)
	ByOwnerModelProviderFunc func(ownerCharacterId uint32) model.Provider[[]door.Model]
	GetByOwnerFunc           func(ownerCharacterId uint32) ([]door.Model, error)
	ForEachInMapFunc         func(f field.Model, o model.Operator[door.Model]) error
	GetByOwnerOnMapFunc      func(f field.Model, ownerCharacterId uint32) (door.Model, bool)
	SpawnFunc                func(f field.Model, ownerCharacterId, skillId uint32, level byte, x, y int16) error
	RemoveFunc               func(f field.Model, ownerCharacterId uint32, reason string) error
}

var _ door.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) InFieldModelProvider(f field.Model) model.Provider[[]door.Model] {
	if m.InFieldModelProviderFunc != nil {
		return m.InFieldModelProviderFunc(f)
	}
	return model.FixedProvider([]door.Model{})
}

func (m *ProcessorMock) GetInField(f field.Model) ([]door.Model, error) {
	if m.GetInFieldFunc != nil {
		return m.GetInFieldFunc(f)
	}
	return nil, nil
}

func (m *ProcessorMock) ByOwnerModelProvider(ownerCharacterId uint32) model.Provider[[]door.Model] {
	if m.ByOwnerModelProviderFunc != nil {
		return m.ByOwnerModelProviderFunc(ownerCharacterId)
	}
	return model.FixedProvider([]door.Model{})
}

func (m *ProcessorMock) GetByOwner(ownerCharacterId uint32) ([]door.Model, error) {
	if m.GetByOwnerFunc != nil {
		return m.GetByOwnerFunc(ownerCharacterId)
	}
	return nil, nil
}

func (m *ProcessorMock) ForEachInMap(f field.Model, o model.Operator[door.Model]) error {
	if m.ForEachInMapFunc != nil {
		return m.ForEachInMapFunc(f, o)
	}
	return nil
}

func (m *ProcessorMock) GetByOwnerOnMap(f field.Model, ownerCharacterId uint32) (door.Model, bool) {
	if m.GetByOwnerOnMapFunc != nil {
		return m.GetByOwnerOnMapFunc(f, ownerCharacterId)
	}
	return door.Model{}, false
}

func (m *ProcessorMock) Spawn(f field.Model, ownerCharacterId, skillId uint32, level byte, x, y int16) error {
	if m.SpawnFunc != nil {
		return m.SpawnFunc(f, ownerCharacterId, skillId, level, x, y)
	}
	return nil
}

func (m *ProcessorMock) Remove(f field.Model, ownerCharacterId uint32, reason string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(f, ownerCharacterId, reason)
	}
	return nil
}
