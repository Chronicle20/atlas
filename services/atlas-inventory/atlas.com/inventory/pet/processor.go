package pet

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	ByIdProvider(petId uint32) model.Provider[Model]
	GetById(petId uint32) (Model, error)
	Create(characterId uint32, templateId uint32) (Model, error)
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

func (p *ProcessorImpl) ByIdProvider(petId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(petId), Extract)
}

func (p *ProcessorImpl) GetById(petId uint32) (Model, error) {
	return p.ByIdProvider(petId)()
}

func (p *ProcessorImpl) Create(characterId uint32, templateId uint32) (Model, error) {
	i := Model{
		ownerId:    characterId,
		templateId: templateId,
	}
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestCreate(i), Extract)()
}
