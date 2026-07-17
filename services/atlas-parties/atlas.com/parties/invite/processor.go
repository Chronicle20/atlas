package invite

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Processor interface {
	Create(actorId uint32, worldId world.Id, partyId uint32, targetId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		p:   producer.ProviderImpl(l)(ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Create(actorId uint32, worldId world.Id, partyId uint32, targetId uint32) error {
	p.l.Debugf("Creating party [%d] invitation for [%d] from [%d].", partyId, targetId, actorId)
	return p.p(EnvCommandTopic)(createInviteCommandProvider(actorId, partyId, worldId, targetId))
}
