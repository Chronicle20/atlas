package mock

import (
	"atlas-pets/character"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
)

type Processor struct {
	GetByIdFn            func(...model.Decorator[character.Model]) func(uint32) (character.Model, error)
	InventoryDecoratorFn func(character.Model) character.Model
	EnterFn              func(field field.Model, characterId uint32)
	ExitFn               func(field field.Model, characterId uint32)
	TransitionMapFn      func(field field.Model, characterId uint32, oldMapId _map.Id)
	TransitionChannelFn  func(field field.Model, characterId uint32, oldChannelId channel.Id)
}

func (m *Processor) GetById(d ...model.Decorator[character.Model]) func(uint32) (character.Model, error) {
	return m.GetByIdFn(d...)
}

func (m *Processor) InventoryDecorator(c character.Model) character.Model {
	return m.InventoryDecoratorFn(c)
}

func (m *Processor) Enter(field field.Model, characterId uint32) {
	m.EnterFn(field, characterId)
}

func (m *Processor) Exit(field field.Model, characterId uint32) {
	m.ExitFn(field, characterId)
}

func (m *Processor) TransitionMap(field field.Model, characterId uint32, oldMapId _map.Id) {
	m.TransitionMapFn(field, characterId, oldMapId)
}

func (m *Processor) TransitionChannel(field field.Model, characterId uint32, oldChannelId channel.Id) {
	m.TransitionChannelFn(field, characterId, oldChannelId)
}
