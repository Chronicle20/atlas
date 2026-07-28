package mock

import (
	"atlas-doors/door"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

type ProcessorMock struct {
	GetByIdFunc                  func(areaDoorId uint32) (door.Model, error)
	GetInFieldFunc               func(f field.Model) ([]door.Model, error)
	GetByOwnerFunc               func(ownerCharacterId character.Id) ([]door.Model, error)
	SpawnFunc                    func(f field.Model, ownerCharacterId character.Id, skillId skill.Id, skillLevel byte, x point.X, y point.Y) (door.Model, error)
	RemoveByOwnerFunc            func(ownerCharacterId character.Id, reason string) error
	RemoveByOwnerIfLeftFieldFunc func(ownerCharacterId character.Id, newField field.Model) error
	ReslotFunc                   func(areaDoorId uint32, newSlot byte, townPortalId uint32, townX point.X, townY point.Y) error
}

var _ door.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(areaDoorId uint32) (door.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(areaDoorId)
	}
	return door.Model{}, nil
}

func (m *ProcessorMock) GetInField(f field.Model) ([]door.Model, error) {
	if m.GetInFieldFunc != nil {
		return m.GetInFieldFunc(f)
	}
	return []door.Model{}, nil
}

func (m *ProcessorMock) GetByOwner(ownerCharacterId character.Id) ([]door.Model, error) {
	if m.GetByOwnerFunc != nil {
		return m.GetByOwnerFunc(ownerCharacterId)
	}
	return []door.Model{}, nil
}

func (m *ProcessorMock) Spawn(f field.Model, ownerCharacterId character.Id, skillId skill.Id, skillLevel byte, x point.X, y point.Y) (door.Model, error) {
	if m.SpawnFunc != nil {
		return m.SpawnFunc(f, ownerCharacterId, skillId, skillLevel, x, y)
	}
	return door.Model{}, nil
}

func (m *ProcessorMock) RemoveByOwner(ownerCharacterId character.Id, reason string) error {
	if m.RemoveByOwnerFunc != nil {
		return m.RemoveByOwnerFunc(ownerCharacterId, reason)
	}
	return nil
}

func (m *ProcessorMock) RemoveByOwnerIfLeftField(ownerCharacterId character.Id, newField field.Model) error {
	if m.RemoveByOwnerIfLeftFieldFunc != nil {
		return m.RemoveByOwnerIfLeftFieldFunc(ownerCharacterId, newField)
	}
	return nil
}

func (m *ProcessorMock) Reslot(areaDoorId uint32, newSlot byte, townPortalId uint32, townX point.X, townY point.Y) error {
	if m.ReslotFunc != nil {
		return m.ReslotFunc(areaDoorId, newSlot, townPortalId, townX, townY)
	}
	return nil
}
