package monster

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Processor provides monster spawning functionality.
type Processor interface {
	// SpawnMonster spawns a monster at the location described by req.
	SpawnMonster(f field.Model, req SpawnRequest) error
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

func (p *ProcessorImpl) SpawnMonster(f field.Model, req SpawnRequest) error {
	_, err := requestSpawnMonster(p.ctx, f, req.ToRestModel())(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to spawn monster %d at (%d, %d) in map %d", req.MonsterId, req.X, req.Y, f.MapId())
		return err
	}

	p.l.Debugf("Successfully spawned monster %d at (%d, %d, fh=%d) in world %d, channel %d, map %d",
		req.MonsterId, req.X, req.Y, req.Fh, f.WorldId(), f.ChannelId(), f.MapId())
	return nil
}
