package mapdata

import (
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor is the map REST client used by the mini-game validation ladder.
// FieldLimit backs the "cannot start game here" check (bit 0x80).
type Processor interface {
	GetById(mapId _map.Id) (Model, error)
	ByIdProvider(mapId _map.Id) model.Provider[Model]
	FieldLimit(mapId _map.Id) (uint32, error)
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ByIdProvider(mapId _map.Id) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestMap(mapId), Extract)
}

func (p *ProcessorImpl) GetById(mapId _map.Id) (Model, error) {
	return p.ByIdProvider(mapId)()
}

func (p *ProcessorImpl) FieldLimit(mapId _map.Id) (uint32, error) {
	m, err := p.GetById(mapId)
	if err != nil {
		return 0, err
	}
	return m.FieldLimit(), nil
}
