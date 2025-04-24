package position

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
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

func (p *Processor) GetBelow(mapId uint32, x int16, y int16) model.Provider[Model] {
	return requests.Provider[FootholdRestModel, Model](p.l, p.ctx)(getInMap(mapId, x, y), Extract)
}
