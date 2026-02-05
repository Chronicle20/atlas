package mock

import (
	"atlas-pets/character"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
)

type Processor struct {
	GetByIdFn            func(...model.Decorator[character.Model]) func(uint32) (character.Model, error)
	InventoryDecoratorFn func(character.Model) character.Model
	EnterFn              func(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32)
	ExitFn               func(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32)
	TransitionMapFn      func(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id)
	TransitionChannelFn  func(worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id)
}

func (m *Processor) GetById(d ...model.Decorator[character.Model]) func(uint32) (character.Model, error) {
	return m.GetByIdFn(d...)
}

func (m *Processor) InventoryDecorator(c character.Model) character.Model {
	return m.InventoryDecoratorFn(c)
}

func (m *Processor) Enter(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) {
	m.EnterFn(worldId, channelId, mapId, characterId)
}

func (m *Processor) Exit(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) {
	m.ExitFn(worldId, channelId, mapId, characterId)
}

func (m *Processor) TransitionMap(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id) {
	m.TransitionMapFn(worldId, channelId, mapId, characterId, oldMapId)
}

func (m *Processor) TransitionChannel(worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id) {
	m.TransitionChannelFn(worldId, channelId, oldChannelId, characterId, mapId)
}
