package chair

import (
	chair2 "atlas-channel/kafka/message/chair"
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for chair processing
type Processor interface {
	InMapModelProvider(f field.Model) model.Provider[[]Model]
	ForEachInMap(f field.Model, o model.Operator[Model]) error
	Use(f field.Model, chairType string, chairId uint32, characterId uint32) error
	Cancel(f field.Model, characterId uint32) error
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

func (p *ProcessorImpl) Use(f field.Model, chairType string, chairId uint32, characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to use map [%d] [%s] chair [%d].", characterId, f.MapId(), chairType, chairId)
	return producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvCommandTopic)(UseCommandProvider(f, chairType, chairId, characterId))
}

func (p *ProcessorImpl) Cancel(f field.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvCommandTopic)(CancelCommandProvider(f, characterId))
}
