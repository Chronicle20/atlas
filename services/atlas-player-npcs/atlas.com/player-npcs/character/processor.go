package character

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetById(characterId uint32) (Model, error)
	ByNameProvider(name string) model.Provider[[]Model]
	GetByName(name string) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(p.ctx, characterId), Extract)()
}

func (p *ProcessorImpl) ByNameProvider(name string) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByName(p.ctx, name), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByName(name string) (Model, error) {
	return model.First(p.ByNameProvider(name), model.Filters[Model]())
}
