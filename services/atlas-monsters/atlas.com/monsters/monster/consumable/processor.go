package consumable

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetById(itemId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetById resolves a catch item's WZ data for the request's tenant. There is
// deliberately NO cache: 0227.img differs by region and version and the
// reward ids differ with it, catch attempts are rare, and a tenant-keyed
// cache would buy nothing for the risk of serving one tenant another's
// reward id (task-212 design §7).
func (p *ProcessorImpl) GetById(itemId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(p.ctx, itemId), Extract)()
}
