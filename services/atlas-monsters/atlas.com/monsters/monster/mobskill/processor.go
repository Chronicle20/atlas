package mobskill

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetByIdAndLevel(skillId uint16, level uint16) (Model, error)
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

func (p *ProcessorImpl) GetByIdAndLevel(skillId uint16, level uint16) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestByIdAndLevel(skillId, level), Extract)()
}
