package npc_spawn

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Processor provides NPC placement functionality against atlas-maps.
type Processor interface {
	// SpawnNpc places an NPC at the location described by req.
	SpawnNpc(f field.Model, req SpawnRequest) error
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

func (p *ProcessorImpl) SpawnNpc(f field.Model, req SpawnRequest) error {
	_, err := requestSpawnNpc(p.ctx, f, req.ToRestModel())(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to spawn npc %d at (%d, %d) in map %d", req.NpcId, req.X, req.Y, f.MapId())
		return err
	}

	p.l.Debugf("Successfully spawned npc %d at (%d, %d, fh=%d) in world %d, channel %d, map %d",
		req.NpcId, req.X, req.Y, req.Fh, f.WorldId(), f.ChannelId(), f.MapId())
	return nil
}
