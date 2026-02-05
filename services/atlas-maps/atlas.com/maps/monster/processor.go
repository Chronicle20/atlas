package monster

import (
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	CountInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) (int, error)
	CreateMonster(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, monsterId uint32, x int16, y int16, fh uint16, team int32)
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

func (p *ProcessorImpl) CountInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) (int, error) {
	data, err := requestInMap(worldId, channelId, mapId)(p.l, p.ctx)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (p *ProcessorImpl) CreateMonster(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, monsterId uint32, x int16, y int16, fh uint16, team int32) {
	_, err := requestCreate(worldId, channelId, mapId, monsterId, x, y, fh, team)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Creating monster for map %d", mapId)
	}
}
