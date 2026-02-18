package monster

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	DestroyInField(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error
	SpawnInField(f field.Model, monsterId uint32, x int16, y int16, fh int16) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) DestroyInField(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error {
	return requestDestroyInField(worldId, channelId, mapId, instance)(p.l, p.ctx)
}

func (p *ProcessorImpl) SpawnInField(f field.Model, monsterId uint32, x int16, y int16, fh int16) error {
	input := SpawnInputRestModel{
		Id:        "0",
		MonsterId: monsterId,
		X:         x,
		Y:         y,
		Fh:        fh,
		Team:      0,
	}
	_, err := requestSpawnInField(f, input)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to spawn monster [%d] in field [%s].", monsterId, f.Id())
		return err
	}
	p.l.Debugf("Spawned monster [%d] at (%d, %d) in field [%s].", monsterId, x, y, f.Id())
	return nil
}
