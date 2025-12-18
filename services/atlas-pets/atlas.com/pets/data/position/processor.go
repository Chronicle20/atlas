package position

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetBelow(mapId uint32, x int16, y int16) model.Provider[Model]
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

func (p *ProcessorImpl) GetBelow(mapId uint32, x int16, y int16) model.Provider[Model] {
	return requests.Provider[FootholdRestModel, Model](p.l, p.ctx)(getInMap(mapId, x, y), Extract)
}
