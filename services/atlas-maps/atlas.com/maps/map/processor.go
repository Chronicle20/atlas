package _map

import (
	"atlas-maps/kafka/message"
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/kafka/producer"
	"atlas-maps/map/character"
	monster2 "atlas-maps/map/monster"
	"atlas-maps/reactor"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Enter(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
	EnterAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
	Exit(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
	ExitAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
	TransitionMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id)
	TransitionMapAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id) error
	TransitionChannel(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id)
	TransitionChannelAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id) error
	GetCharactersInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	p   producer.Provider
	cp  character.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, p producer.Provider) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		p:   p,
		cp:  character.NewProcessor(l, ctx),
	}
}

func (p *ProcessorImpl) Enter(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
	return func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
		p.cp.Enter(transactionId, worldId, channelId, mapId, characterId)
		go func() {
			_ = monster2.NewProcessor(p.l, p.ctx).SpawnMonsters(transactionId)(worldId)(channelId)(mapId)
		}()
		go func() {
			_ = reactor.NewProcessor(p.l, p.ctx, p.p).SpawnAndEmit(transactionId, worldId, channelId, mapId)
		}()
		return mb.Put(mapKafka.EnvEventTopicMapStatus, enterMapProvider(transactionId, worldId, channelId, mapId, characterId))
	}
}

func (p *ProcessorImpl) EnterAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Enter(buf)(transactionId, worldId, channelId, mapId, characterId)
	})
}

func (p *ProcessorImpl) Exit(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
	return func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
		p.cp.Exit(transactionId, worldId, channelId, mapId, characterId)
		return mb.Put(mapKafka.EnvEventTopicMapStatus, exitMapProvider(transactionId, worldId, channelId, mapId, characterId))
	}
}

func (p *ProcessorImpl) ExitAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Exit(buf)(transactionId, worldId, channelId, mapId, characterId)
	})
}

func (p *ProcessorImpl) TransitionMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id) {
	return func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id) {
		_ = p.Exit(mb)(transactionId, worldId, channelId, oldMapId, characterId)
		_ = p.Enter(mb)(transactionId, worldId, channelId, mapId, characterId)
	}
}

func (p *ProcessorImpl) TransitionMapAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		p.TransitionMap(buf)(transactionId, worldId, channelId, mapId, characterId, oldMapId)
		return nil
	})
}

func (p *ProcessorImpl) TransitionChannel(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id) {
	return func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id) {
		_ = p.Exit(mb)(transactionId, worldId, oldChannelId, mapId, characterId)
		_ = p.Enter(mb)(transactionId, worldId, channelId, mapId, characterId)
	}
}

func (p *ProcessorImpl) TransitionChannelAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		p.TransitionChannel(buf)(transactionId, worldId, channelId, oldChannelId, characterId, mapId)
		return nil
	})
}

func (p *ProcessorImpl) GetCharactersInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error) {
	return p.cp.GetCharactersInMap(transactionId, worldId, channelId, mapId)
}
