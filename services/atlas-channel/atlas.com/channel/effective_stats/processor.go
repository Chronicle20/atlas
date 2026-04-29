package effective_stats

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

// Processor wraps the effective-stats REST client so callers can fetch a
// character's session-effective stats by id.
type Processor interface {
	GetByCharacterId(worldId world.Id, channelId channel.Id, characterId uint32) (RestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) GetByCharacterId(worldId world.Id, channelId channel.Id, characterId uint32) (RestModel, error) {
	return requestByCharacter(worldId, channelId, characterId)(p.l, p.ctx)
}
