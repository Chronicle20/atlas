package reactor

import (
	reactor2 "atlas-maps/data/map/reactor"
	"atlas-maps/kafka/message"
	reactorKafka "atlas-maps/kafka/message/reactor"
	"atlas-maps/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	InMapModelProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]Model]
	GetInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]Model, error)
	Spawn(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error
	SpawnAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, p producer.Provider) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		p:   p,
	}
}

func (p *ProcessorImpl) InMapModelProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMap(byte(worldId), byte(channelId), uint32(mapId)), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]Model, error) {
	return p.InMapModelProvider(transactionId, worldId, channelId, mapId)()
}

func (p *ProcessorImpl) doesNotExist(existing []Model) model.Filter[reactor2.Model] {
	return func(reference reactor2.Model) bool {
		for _, er := range existing {
			if er.Classification() == reference.Classification() && er.X() == reference.X() && er.Y() == reference.Y() {
				return false
			}
		}
		return true
	}
}

func (p *ProcessorImpl) Spawn(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error {
	return func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error {
		existing, err := p.GetInMap(transactionId, worldId, channelId, mapId)
		if err != nil {
			return err
		}

		rp := reactor2.NewProcessor(p.l, p.ctx).InMapProvider(mapId)
		np := model.FilteredProvider(rp, model.Filters[reactor2.Model](p.doesNotExist(existing)))
		return model.ForEachSlice(np, p.issueCreate(mb)(transactionId, worldId, channelId, mapId, instance), model.ParallelExecute())
	}
}

func (p *ProcessorImpl) SpawnAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Spawn(buf)(transactionId, worldId, channelId, mapId, instance)
	})
}

func (p *ProcessorImpl) issueCreate(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) model.Operator[reactor2.Model] {
	return func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) model.Operator[reactor2.Model] {
		return func(r reactor2.Model) error {
			return mb.Put(reactorKafka.EnvCommandTopic, createCommandProvider(transactionId, worldId, channelId, mapId, instance, r.Classification(), r.Name(), 0, r.X(), r.Y(), r.Delay(), r.Direction()))
		}
	}
}
