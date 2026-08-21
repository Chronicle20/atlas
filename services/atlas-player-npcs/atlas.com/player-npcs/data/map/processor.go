package map_

import (
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetById(mapId _map.Id) (Model, error)
	// Ground resolves the highest foothold beneath each point (design
	// §5.3, D-2), preserving request order in the result.
	Ground(mapId _map.Id, points []GroundPoint) ([]GroundResult, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(mapId _map.Id) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(p.ctx, mapId), Extract)()
}

func (p *ProcessorImpl) Ground(mapId _map.Id, points []GroundPoint) ([]GroundResult, error) {
	rmPoints := make([]GroundPointRestModel, 0, len(points))
	for _, pt := range points {
		rmPoints = append(rmPoints, GroundPointRestModel{X: pt.X(), Y: pt.Y()})
	}
	return requests.SliceProvider[GroundResultRestModel, GroundResult](p.l, p.ctx)(requestGround(p.ctx, mapId, rmPoints), ExtractGroundResult, model.Filters[GroundResult]())()
}
