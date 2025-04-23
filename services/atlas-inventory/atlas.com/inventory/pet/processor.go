package pet

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) ByIdProvider(petId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(petId), Extract)
}

func (p *Processor) GetById(petId uint32) (Model, error) {
	return p.ByIdProvider(petId)()
}

func (p *Processor) Create(characterId uint32, templateId uint32) (Model, error) {
	i := Model{
		ownerId:    characterId,
		templateId: templateId,
	}
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestCreate(i), Extract)()
}
