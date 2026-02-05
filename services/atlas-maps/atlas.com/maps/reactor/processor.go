package reactor

import (
	reactor2 "atlas-maps/data/map/reactor"
	"atlas-maps/kafka/message"
	reactorKafka "atlas-maps/kafka/message/reactor"
	"atlas-maps/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	InMapModelProvider(transactionId uuid.UUID, field field.Model) model.Provider[[]Model]
	GetInMap(transactionId uuid.UUID, field field.Model) ([]Model, error)
	Spawn(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model) error
	SpawnAndEmit(transactionId uuid.UUID, field field.Model) error
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

func (p *ProcessorImpl) InMapModelProvider(transactionId uuid.UUID, field field.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMap(field), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetInMap(transactionId uuid.UUID, field field.Model) ([]Model, error) {
	return p.InMapModelProvider(transactionId, field)()
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

func (p *ProcessorImpl) Spawn(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model) error {
	return func(transactionId uuid.UUID, field field.Model) error {
		existing, err := p.GetInMap(transactionId, field)
		if err != nil {
			return err
		}

		rp := reactor2.NewProcessor(p.l, p.ctx).InMapProvider(field.MapId())
		np := model.FilteredProvider(rp, model.Filters[reactor2.Model](p.doesNotExist(existing)))
		return model.ForEachSlice(np, p.issueCreate(mb)(transactionId, field), model.ParallelExecute())
	}
}

func (p *ProcessorImpl) SpawnAndEmit(transactionId uuid.UUID, field field.Model) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Spawn(buf)(transactionId, field)
	})
}

func (p *ProcessorImpl) issueCreate(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model) model.Operator[reactor2.Model] {
	return func(transactionId uuid.UUID, field field.Model) model.Operator[reactor2.Model] {
		return func(r reactor2.Model) error {
			return mb.Put(reactorKafka.EnvCommandTopic, createCommandProvider(transactionId, field, r.Classification(), r.Name(), 0, r.X(), r.Y(), r.Delay(), r.Direction()))
		}
	}
}
