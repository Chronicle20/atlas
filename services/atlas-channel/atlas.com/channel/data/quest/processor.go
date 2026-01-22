package quest

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(questId uint32) (Model, error)
	GetAll() ([]Model, error)
	GetAutoStart() ([]Model, error)
	ByIdProvider(questId uint32) model.Provider[Model]
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) ByIdProvider(questId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(questId), Extract)
}

func (p *ProcessorImpl) GetById(questId uint32) (Model, error) {
	return p.ByIdProvider(questId)()
}

func (p *ProcessorImpl) GetAll() ([]Model, error) {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestAll(), Extract, model.Filters[Model]())()
}

func (p *ProcessorImpl) GetAutoStart() ([]Model, error) {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestAutoStart(), Extract, model.Filters[Model]())()
}
