package foothold

import (
	"context"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

// Processor provides foothold lookup functionality.
type Processor interface {
	// GetFootholdBelow looks up the foothold ID below a given position in a map.
	// Returns the foothold ID if found, or 0 if no foothold exists at the position.
	GetFootholdBelow(mapId _map.Id, x, y int16) (uint32, error)
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

func (p *ProcessorImpl) GetFootholdBelow(mapId _map.Id, x, y int16) (uint32, error) {
	input := PositionInputRestModel{
		X: x,
		Y: y,
	}

	result, err := requestFootholdBelow(mapId, input)(p.l, p.ctx)
	if err != nil {
		// If no foothold is found, atlas-data returns an error (500 status)
		// Return 0 as the default foothold ID
		p.l.WithError(err).Debugf("Failed to get foothold below position (%d, %d) in map %d, using default fh=0", x, y, mapId)
		return 0, nil
	}

	return result.Id, nil
}
