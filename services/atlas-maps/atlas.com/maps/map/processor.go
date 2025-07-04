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
	Enter(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32)
	EnterAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
	Exit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32)
	ExitAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
	TransitionMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id)
	TransitionMapAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id) error
	TransitionChannel(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id)
	TransitionChannelAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id) error
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

func (p *ProcessorImpl) Enter(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) {
	p.cp.Enter(transactionId, worldId, channelId, mapId, characterId)
	go func() {
		_ = monster2.NewProcessor(p.l, p.ctx).SpawnMonsters(transactionId)(worldId)(channelId)(mapId)
	}()
	go func() {
		_ = reactor.NewProcessor(p.l, p.ctx, p.p).SpawnAndEmit(transactionId, worldId, channelId, mapId)
	}()
}

func (p *ProcessorImpl) EnterAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		p.Enter(transactionId, worldId, channelId, mapId, characterId)
		return buf.Put(mapKafka.EnvEventTopicMapStatus, enterMapProvider(transactionId, worldId, channelId, mapId, characterId))
	})
}

func (p *ProcessorImpl) Exit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) {
	p.cp.Exit(transactionId, worldId, channelId, mapId, characterId)
}

func (p *ProcessorImpl) ExitAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		p.Exit(transactionId, worldId, channelId, mapId, characterId)
		return buf.Put(mapKafka.EnvEventTopicMapStatus, exitMapProvider(transactionId, worldId, channelId, mapId, characterId))
	})
}

func (p *ProcessorImpl) TransitionMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id) {
	p.Exit(transactionId, worldId, channelId, oldMapId, characterId)
	p.Enter(transactionId, worldId, channelId, mapId, characterId)
}

func (p *ProcessorImpl) TransitionMapAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		p.TransitionMap(transactionId, worldId, channelId, mapId, characterId, oldMapId)
		return nil
	})
}

func (p *ProcessorImpl) TransitionChannel(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id) {
	p.Exit(transactionId, worldId, oldChannelId, mapId, characterId)
	p.Enter(transactionId, worldId, channelId, mapId, characterId)
}

func (p *ProcessorImpl) TransitionChannelAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		p.TransitionChannel(transactionId, worldId, channelId, oldChannelId, characterId, mapId)
		return nil
	})
}
