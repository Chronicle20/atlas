package chalkboard

import (
	chalkboard2 "atlas-channel/kafka/message/chalkboard"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for chalkboard processing
type Processor interface {
	InMapModelProvider(f field.Model) model.Provider[[]Model]
	ForEachInMap(f field.Model, o model.Operator[Model]) error
	AttemptUse(f field.Model, characterId uint32, message string) error
	Close(f field.Model, characterId uint32) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *ProcessorImpl) InMapModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMap(f), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) ForEachInMap(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(f), o, model.ParallelExecute())
}

func (p *ProcessorImpl) AttemptUse(f field.Model, characterId uint32, message string) error {
	p.l.Debugf("Character [%d] attempting to set a chalkboard message [%s].", characterId, message)
	return producer.ProviderImpl(p.l)(p.ctx)(chalkboard2.EnvCommandTopic)(SetCommandProvider(f, characterId, message))
}

func (p *ProcessorImpl) Close(f field.Model, characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to close chalkboard.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(chalkboard2.EnvCommandTopic)(ClearCommandProvider(f, characterId))
}
