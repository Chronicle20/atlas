package drop

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetByMonsterId(monsterId uint32) ([]Model, error)
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetByMonsterId(monsterId uint32) ([]Model, error) {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestForMonster(monsterId), Extract, model.Filters[Model]())()
}
