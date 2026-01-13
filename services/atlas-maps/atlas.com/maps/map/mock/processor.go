package mock

import (
	"atlas-maps/kafka/message"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type Processor struct {
	EnterFunc                  func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
	EnterAndEmitFunc           func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
	ExitFunc                   func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
	ExitAndEmitFunc            func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
	TransitionMapFunc          func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id)
	TransitionMapAndEmitFunc   func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id) error
	TransitionChannelFunc      func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id)
	TransitionChannelAndEmitFunc func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id) error
	GetCharactersInMapFunc     func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error)
}

func (m *Processor) Enter(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
	if m.EnterFunc != nil {
		return m.EnterFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
		return nil
	}
}

func (m *Processor) EnterAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
	if m.EnterAndEmitFunc != nil {
		return m.EnterAndEmitFunc(transactionId, worldId, channelId, mapId, characterId)
	}
	return nil
}

func (m *Processor) Exit(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
	if m.ExitFunc != nil {
		return m.ExitFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
		return nil
	}
}

func (m *Processor) ExitAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
	if m.ExitAndEmitFunc != nil {
		return m.ExitAndEmitFunc(transactionId, worldId, channelId, mapId, characterId)
	}
	return nil
}

func (m *Processor) TransitionMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id) {
	if m.TransitionMapFunc != nil {
		return m.TransitionMapFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id) {
	}
}

func (m *Processor) TransitionMapAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id) error {
	if m.TransitionMapAndEmitFunc != nil {
		return m.TransitionMapAndEmitFunc(transactionId, worldId, channelId, mapId, characterId, oldMapId)
	}
	return nil
}

func (m *Processor) TransitionChannel(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id) {
	if m.TransitionChannelFunc != nil {
		return m.TransitionChannelFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id) {
	}
}

func (m *Processor) TransitionChannelAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id) error {
	if m.TransitionChannelAndEmitFunc != nil {
		return m.TransitionChannelAndEmitFunc(transactionId, worldId, channelId, oldChannelId, characterId, mapId)
	}
	return nil
}

func (m *Processor) GetCharactersInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error) {
	if m.GetCharactersInMapFunc != nil {
		return m.GetCharactersInMapFunc(transactionId, worldId, channelId, mapId)
	}
	return nil, nil
}
