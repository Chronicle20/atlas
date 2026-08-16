package foothold

import (
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor resolves where a dropped object comes to rest.
type Processor interface {
	// LandingBelow returns the y-coordinate of the floor directly beneath
	// (x, y) in the given map — the foothold a straight-down drop lands on.
	// ok is false when no foothold sits below the point (e.g. the map edge
	// or a bottomless pit); callers should keep the original y in that case.
	LandingBelow(mapId _map.Id, x int16, y int16) (int16, bool)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) LandingBelow(mapId _map.Id, x int16, y int16) (int16, bool) {
	fh, err := requests.Provider[FootholdRestModel, Model](p.l, p.ctx)(requestBelow(p.ctx, mapId, x, y), Extract)()
	if err != nil {
		// atlas-data returns an error status when nothing is below the point.
		// Not exceptional — the caller falls back to the original y.
		p.l.WithError(err).Debugf("No foothold below position (%d, %d) in map %d; leaving drop y unchanged.", x, y, mapId)
		return 0, false
	}
	return fh.LandingY(x)
}
