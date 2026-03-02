package pet

import (
	"context"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Create(ownerId uint32, templateId uint32, name string) (Model, error)
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

func (p *ProcessorImpl) Create(ownerId uint32, templateId uint32, name string) (Model, error) {
	i := Model{
		ownerId:    ownerId,
		templateId: templateId,
		name:       name,
	}
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestCreate(i), Extract)()
}
