package mock

import (
	"atlas-pets/character"
	"github.com/Chronicle20/atlas-model/model"
)

type Processor struct {
	GetByIdFn            func(...model.Decorator[character.Model]) func(uint32) (character.Model, error)
	InventoryDecoratorFn func(character.Model) character.Model
	EnterFn              func(worldId, channelId byte, mapId uint32, characterId uint32)
	ExitFn               func(worldId, channelId byte, mapId uint32, characterId uint32)
	TransitionMapFn      func(worldId, channelId byte, mapId uint32, characterId uint32, oldMapId uint32)
	TransitionChannelFn  func(worldId, channelId, oldChannelId byte, characterId uint32, mapId uint32)
}

func (m *Processor) GetById(d ...model.Decorator[character.Model]) func(uint32) (character.Model, error) {
	return m.GetByIdFn(d...)
}

func (m *Processor) InventoryDecorator(c character.Model) character.Model {
	return m.InventoryDecoratorFn(c)
}

func (m *Processor) Enter(worldId, channelId byte, mapId, characterId uint32) {
	m.EnterFn(worldId, channelId, mapId, characterId)
}

func (m *Processor) Exit(worldId, channelId byte, mapId, characterId uint32) {
	m.ExitFn(worldId, channelId, mapId, characterId)
}

func (m *Processor) TransitionMap(worldId, channelId byte, mapId, characterId, oldMapId uint32) {
	m.TransitionMapFn(worldId, channelId, mapId, characterId, oldMapId)
}

func (m *Processor) TransitionChannel(worldId, channelId, oldChannelId byte, characterId, mapId uint32) {
	m.TransitionChannelFn(worldId, channelId, oldChannelId, characterId, mapId)
}
