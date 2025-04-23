package reactor

import (
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) InMapModelProvider(m _map.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMap(m), Extract, model.Filters[Model]())
}

func (p *Processor) ForEachInMap(m _map.Model, f model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(m), f, model.ParallelExecute())
}
