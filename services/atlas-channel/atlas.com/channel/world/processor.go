package world

import (
	"context"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ByIdModelProvider(worldId world.Id) model.Provider[Model]
	GetById(worldId world.Id) (Model, error)
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

func (p *ProcessorImpl) ByIdModelProvider(worldId world.Id) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestWorld(worldId), Extract)
}

func (p *ProcessorImpl) GetById(worldId world.Id) (Model, error) {
	return p.ByIdModelProvider(worldId)()
}
