package monster

import (
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

// Processor provides monster spawning functionality.
type Processor interface {
	// SpawnMonster spawns a monster at the specified location.
	SpawnMonster(worldId world.Id, channelId channel.Id, mapId, monsterId uint32, x, y, fh int16, team int8) error
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

func (p *ProcessorImpl) SpawnMonster(worldId world.Id, channelId channel.Id, mapId, monsterId uint32, x, y, fh int16, team int8) error {
	req := SpawnRequest{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		MonsterId: monsterId,
		X:         x,
		Y:         y,
		Fh:        fh,
		Team:      team,
	}

	_, err := requestSpawnMonster(worldId, channelId, mapId, req.ToRestModel())(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to spawn monster %d at (%d, %d) in map %d", monsterId, x, y, mapId)
		return err
	}

	p.l.Debugf("Successfully spawned monster %d at (%d, %d, fh=%d) in world %d, channel %d, map %d",
		monsterId, x, y, fh, worldId, channelId, mapId)
	return nil
}
