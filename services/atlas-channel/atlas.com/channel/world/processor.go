package world

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	ByIdModelProvider(worldId world.Id) model.Provider[Model]
	GetById(worldId world.Id) (Model, error)
	AllProvider() model.Provider[[]Model]
	GetAll() ([]Model, error)
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

func (p *ProcessorImpl) ByIdModelProvider(worldId world.Id) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestWorld(worldId), Extract)
}

func (p *ProcessorImpl) GetById(worldId world.Id) (Model, error) {
	return p.ByIdModelProvider(worldId)()
}

// AllProvider fetches every world for the tenant. atlas-world's worlds list
// is paginated (task-117), so this drains every page rather than fetching
// just the first — the cash-shop world-transfer name list is indexed by world
// id on the wire, so a truncated list silently misroutes a transfer.
func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(worldsUrl(), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetAll() ([]Model, error) {
	return p.AllProvider()()
}
