package character

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Model struct {
	id   uint32
	name string
}

func (m Model) Id() uint32   { return m.id }
func (m Model) Name() string { return m.name }

type Processor interface {
	GetById(characterId uint32) (Model, error)
	GetByName(name string) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)()
}

func (p *ProcessorImpl) GetByName(name string) (Model, error) {
	ps := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByName(name), Extract, model.Filters[Model]())
	return model.First(ps, model.Filters[Model]())
}
