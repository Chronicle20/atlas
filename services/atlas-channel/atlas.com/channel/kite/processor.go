package kite

import (
	kite2 "atlas-channel/kafka/message/kite"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor interface defines the operations for kite processing.
type Processor interface {
	InMapModelProvider(f field.Model) model.Provider[[]Model]
	ForEachInMap(f field.Model, o model.Operator[Model]) error
	AttemptUse(f field.Model, characterId uint32, name string, templateId uint32, message string, x int16, y int16) error
}

// ProcessorImpl implements the Processor interface.
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

var _ Processor = (*ProcessorImpl)(nil)

// InMapModelProvider fetches every kite currently placed in one map instance
// (used to replay existing kite state to a character entering the map). The
// upstream atlas-kites list is paginated, so this drains every page rather
// than fetching just the first.
func (p *ProcessorImpl) InMapModelProvider(f field.Model) model.Provider[[]Model] {
	url, err := inMapUrl(p.ctx, f)
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) ForEachInMap(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(f), o, model.ParallelExecute())
}

func (p *ProcessorImpl) AttemptUse(f field.Model, characterId uint32, name string, templateId uint32, message string, x int16, y int16) error {
	p.l.Debugf("Character [%d] attempting to place kite [%s] with template [%d].", characterId, name, templateId)
	return producer.ProviderImpl(p.l)(p.ctx)(kite2.EnvCommandTopic)(CreateCommandProvider(f, characterId, name, templateId, message, x, y))
}
