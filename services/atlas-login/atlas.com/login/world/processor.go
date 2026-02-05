package world

import (
	"atlas-login/channel"
	"context"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
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

func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestWorlds(), Extract, model.Filters[Model]())
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
