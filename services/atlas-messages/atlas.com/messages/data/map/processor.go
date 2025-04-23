package _map

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
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) GetById(mapId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestMap(mapId), Extract)()
}
