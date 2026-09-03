package npc

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor interface defines the operations for scripted-NPC read access.
type Processor interface {
	InMapModelProvider(f field.Model) model.Provider[[]Model]
	ForEachInMap(f field.Model, o model.Operator[Model]) error
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

// InMapModelProvider fetches every scripted NPC currently placed on one
// map instance (used to replay existing state to a character entering the
// field). atlas-maps' own list is not paginated, so this is a single
// request through requests.DrainProvider, which treats an unenveloped
// response as the complete collection.
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
