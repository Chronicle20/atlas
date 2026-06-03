package foothold

import (
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetBelow(mapId _map.Id, x int16, y int16) (Model, error)
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

func (p *ProcessorImpl) GetBelow(mapId _map.Id, x int16, y int16) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(getInMap(mapId, x, y), Extract)()
}
