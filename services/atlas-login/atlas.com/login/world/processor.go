package world

import (
	"atlas-login/channel"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetAll() ([]Model, error)
	AllProvider() model.Provider[[]Model]
	GetById(worldId world.Id) (Model, error)
	ByIdModelProvider(worldId world.Id) model.Provider[Model]
	GetCapacityStatus(worldId world.Id) Status
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	cp  channel.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		cp:  channel.NewProcessor(l, ctx),
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

// AllProvider fetches every world for the tenant. The upstream atlas-world
// worlds list is now paginated (task-117); this is a genuine startup
// consumer (server list / world-select screens need every world, not just
// page 1), so it drains every page rather than fetching just the first.
func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(worldsUrl(), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetAll() ([]Model, error) {
	return p.AllProvider()()
}

func (p *ProcessorImpl) ByIdModelProvider(worldId world.Id) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestWorld(worldId), Extract)
}

func (p *ProcessorImpl) GetById(worldId world.Id) (Model, error) {
	return p.ByIdModelProvider(worldId)()
}

func (p *ProcessorImpl) GetCapacityStatus(worldId world.Id) Status {
	w, err := p.GetById(worldId)
	if err != nil {
		return StatusFull
	}
	return w.CapacityStatus()
}
