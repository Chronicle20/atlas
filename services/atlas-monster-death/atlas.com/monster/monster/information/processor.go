package information

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(monsterId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(monsterId uint32) (Model, error) {
	return GetById(p.l)(p.ctx)(monsterId)
}
