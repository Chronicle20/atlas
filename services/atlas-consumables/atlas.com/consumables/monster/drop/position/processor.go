package position

import (
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetInMap(mapId _map.Id, initialX int16, initialY int16, fallbackX int16, fallbackY int16) model.Provider[Model]
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

func (p *ProcessorImpl) GetInMap(mapId _map.Id, initialX int16, initialY int16, fallbackX int16, fallbackY int16) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(getInMap(mapId, initialX, initialY, fallbackX, fallbackY), Extract)
}
