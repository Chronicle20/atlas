package consumable

import (
	"context"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(itemId uint32) (Model, error)
	GetRechargeable() ([]Model, error)
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

func (p *ProcessorImpl) GetById(itemId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(itemId), Extract)()
}

func (p *ProcessorImpl) GetRechargeable() ([]Model, error) {
	restModels, err := requestRechargeable()(p.l, p.ctx)
	if err != nil {
		return nil, err
	}

	models := make([]Model, 0, len(restModels))
	for _, rm := range restModels {
		m, err := Extract(rm)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}

	return models, nil
}
