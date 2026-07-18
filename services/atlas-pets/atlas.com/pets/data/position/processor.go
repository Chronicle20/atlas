package position

import (
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetBelow(mapId _map.Id, x int16, y int16) model.Provider[Model]
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetBelow(mapId _map.Id, x int16, y int16) model.Provider[Model] {
	return requests.Provider[FootholdRestModel, Model](p.l, p.ctx)(getInMap(mapId, x, y), Extract)
}
