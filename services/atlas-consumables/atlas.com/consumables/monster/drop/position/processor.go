package position

import (
	"context"

	_map "github.com/Chronicle20/atlas-constants/map"
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

func (p *Processor) GetInMap(mapId _map.Id, initialX int16, initialY int16, fallbackX int16, fallbackY int16) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(getInMap(mapId, initialX, initialY, fallbackX, fallbackY), Extract)
}
