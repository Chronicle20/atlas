package mock

import (
	"atlas-maps/kafka/message"
	"atlas-maps/reactor"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

type Processor struct {
	InMapModelProviderFunc func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]reactor.Model]
	GetInMapFunc           func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]reactor.Model, error)
	SpawnFunc              func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error
	SpawnAndEmitFunc       func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error
}

func (m *Processor) InMapModelProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]reactor.Model] {
	if m.InMapModelProviderFunc != nil {
		return m.InMapModelProviderFunc(transactionId, worldId, channelId, mapId)
	}
	return func() ([]reactor.Model, error) {
		return nil, nil
	}
}

func (m *Processor) GetInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]reactor.Model, error) {
	if m.GetInMapFunc != nil {
		return m.GetInMapFunc(transactionId, worldId, channelId, mapId)
	}
	return nil, nil
}

func (m *Processor) Spawn(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error {
	if m.SpawnFunc != nil {
		return m.SpawnFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error {
		return nil
	}
}

func (m *Processor) SpawnAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error {
	if m.SpawnAndEmitFunc != nil {
		return m.SpawnAndEmitFunc(transactionId, worldId, channelId, mapId, instance)
	}
	return nil
}
