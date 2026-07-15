package mock

import (
	"atlas-channel/movement"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

type ProcessorMock struct {
	ForCharacterFunc func(f field.Model, characterId uint32, movement model.Movement) error
	ForNPCFunc       func(f field.Model, characterId uint32, objectId uint32, unk byte, unk2 byte, movement model.Movement) error
	ForPetFunc       func(f field.Model, characterId uint32, petId uint32, movement model.Movement) error
	ForMonsterFunc   func(f field.Model, characterId uint32, objectId uint32, moveId int16, skillPossible bool, skill int8, skillId int16, skillLevel int16, mt model.MultiTargetForBall, rt model.RandTimeForAreaAttack, movement model.Movement) error
}

var _ movement.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ForCharacter(f field.Model, characterId uint32, mv model.Movement) error {
	if m.ForCharacterFunc != nil {
		return m.ForCharacterFunc(f, characterId, mv)
	}
	return nil
}

func (m *ProcessorMock) ForNPC(f field.Model, characterId uint32, objectId uint32, unk byte, unk2 byte, mv model.Movement) error {
	if m.ForNPCFunc != nil {
		return m.ForNPCFunc(f, characterId, objectId, unk, unk2, mv)
	}
	return nil
}

func (m *ProcessorMock) ForPet(f field.Model, characterId uint32, petId uint32, mv model.Movement) error {
	if m.ForPetFunc != nil {
		return m.ForPetFunc(f, characterId, petId, mv)
	}
	return nil
}

func (m *ProcessorMock) ForMonster(f field.Model, characterId uint32, objectId uint32, moveId int16, skillPossible bool, skill int8, skillId int16, skillLevel int16, mt model.MultiTargetForBall, rt model.RandTimeForAreaAttack, mv model.Movement) error {
	if m.ForMonsterFunc != nil {
		return m.ForMonsterFunc(f, characterId, objectId, moveId, skillPossible, skill, skillId, skillLevel, mt, rt, mv)
	}
	return nil
}
