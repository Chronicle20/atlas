package effectivestats

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor fetches a character's session-effective combat stats from the
// atlas-effective-stats service. World+channel are required because effective
// stats depend on channel-scoped session context (buffs).
type Processor interface {
	GetByCharacter(worldId world.Id, channelId channel.Id, characterId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) GetByCharacter(worldId world.Id, channelId channel.Id, characterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestByCharacter(worldId, channelId, characterId), Extract)()
}
