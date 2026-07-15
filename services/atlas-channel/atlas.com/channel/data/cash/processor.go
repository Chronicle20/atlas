package cash

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(itemId uint32) (RestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(itemId uint32) (RestModel, error) {
	return requests.Provider[RestModel, RestModel](p.l, p.ctx)(requestById(itemId), Extract)()
}

func Extract(m RestModel) (RestModel, error) {
	return m, nil
}
