package consumable

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(itemId uint32) (Model, error)
	GetRechargeable() ([]Model, error)
}

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

func (p *ProcessorImpl) GetById(itemId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(itemId), Extract)()
}

func (p *ProcessorImpl) GetRechargeable() ([]Model, error) {
	// atlas-data's GET /data/consumables?filter[rechargeable]=true is now
	// paginated (task-117); this drains every page rather than fetching one,
	// since callers need the complete rechargeable set.
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(rechargeableUrl(), 250, Extract, model.Filters[Model]())()
}
