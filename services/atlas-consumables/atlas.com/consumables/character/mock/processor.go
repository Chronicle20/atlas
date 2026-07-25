package mock

import (
	"atlas-consumables/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	GetByIdFunc            func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error)
	InventoryDecoratorFunc func(m character.Model) character.Model
	ChangeMapFunc          func(f field.Model, characterId uint32, portalId uint32) error
	ChangeHPFunc           func(f field.Model, characterId uint32, amount int16) error
	ChangeMPFunc           func(f field.Model, characterId uint32, amount int16) error
}

var _ character.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(decorators...)
	}
	return func(characterId uint32) (character.Model, error) {
		return character.Model{}, nil
	}
}

func (m *ProcessorMock) InventoryDecorator(mo character.Model) character.Model {
	if m.InventoryDecoratorFunc != nil {
		return m.InventoryDecoratorFunc(mo)
	}
	return mo
}

func (m *ProcessorMock) ChangeMap(f field.Model, characterId uint32, portalId uint32) error {
	if m.ChangeMapFunc != nil {
		return m.ChangeMapFunc(f, characterId, portalId)
	}
	return nil
}

func (m *ProcessorMock) ChangeHP(f field.Model, characterId uint32, amount int16) error {
	if m.ChangeHPFunc != nil {
		return m.ChangeHPFunc(f, characterId, amount)
	}
	return nil
}

func (m *ProcessorMock) ChangeMP(f field.Model, characterId uint32, amount int16) error {
	if m.ChangeMPFunc != nil {
		return m.ChangeMPFunc(f, characterId, amount)
	}
	return nil
}
