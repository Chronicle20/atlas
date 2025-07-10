package invite

import (
	"atlas-parties/kafka/producer"
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	Create(actorId uint32, worldId byte, partyId uint32, targetId uint32) error
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

func (p *ProcessorImpl) Create(actorId uint32, worldId byte, partyId uint32, targetId uint32) error {
	p.l.Debugf("Creating party [%d] invitation for [%d] from [%d].", partyId, targetId, actorId)
	return p.p(EnvCommandTopic)(createInviteCommandProvider(actorId, partyId, worldId, targetId))
}
