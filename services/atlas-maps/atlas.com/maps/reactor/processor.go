package reactor

import (
	"atlas-maps/kafka/message"
	reactorKafka "atlas-maps/kafka/message/reactor"
	"atlas-maps/kafka/producer"
	"atlas-maps/map/reactor"
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
	Spawn(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) error
	SpawnAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) error
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

func (p *ProcessorImpl) doesNotExist(existing []Model) model.Filter[reactor.Model] {
	return func(reference reactor.Model) bool {
		for _, er := range existing {
			if er.Classification() == reference.Classification() && er.X() == reference.X() && er.Y() == reference.Y() {
				return false
			}
		}
		return true
	}
}

func (p *ProcessorImpl) Spawn(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) error {
	return func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) error {
		existing, err := p.GetInMap(transactionId, worldId, channelId, mapId)
		if err != nil {
			return err
		}

		rp := reactor.NewProcessor(p.l, p.ctx).InMapProvider(mapId)
		np := model.FilteredProvider(rp, model.Filters[reactor.Model](p.doesNotExist(existing)))
		return model.ForEachSlice(np, p.issueCreate(mb)(transactionId, worldId, channelId, mapId), model.ParallelExecute())
	}
}

func (p *ProcessorImpl) SpawnAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Spawn(buf)(transactionId, worldId, channelId, mapId)
	})
}

func (p *ProcessorImpl) issueCreate(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) model.Operator[reactor.Model] {
	return func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) model.Operator[reactor.Model] {
		return func(r reactor.Model) error {
			return mb.Put(reactorKafka.EnvCommandTopic, createCommandProvider(transactionId, worldId, channelId, mapId, r.Classification(), r.Name(), 0, r.X(), r.Y(), r.Delay(), r.Direction()))
		}
	}
}
