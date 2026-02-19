package cash

import (
	"context"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

func (p *Processor) GetById(itemId uint32) (RestModel, error) {
	return requests.Provider[RestModel, RestModel](p.l, p.ctx)(requestById(itemId), Extract)()
}

func Extract(m RestModel) (RestModel, error) {
	return m, nil
}
