package shops

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ByTemplateIdProvider(templateId uint32) model.Provider[Model]
	GetShop(template uint32) (Model, error)
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

func (p *ProcessorImpl) ByTemplateIdProvider(templateId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestNPCShop(templateId), Extract)
}

func (p *ProcessorImpl) GetShop(template uint32) (Model, error) {
	return p.ByTemplateIdProvider(template)()
}
