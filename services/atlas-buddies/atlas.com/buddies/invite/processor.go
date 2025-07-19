package invite

import (
	invite2 "atlas-buddies/kafka/message/invite"
	"atlas-buddies/kafka/producer"
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	Create(actorId uint32, worldId byte, targetId uint32) error
	Reject(actorId uint32, worldId byte, originatorId uint32) error
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

func (p *ProcessorImpl) Create(actorId uint32, worldId byte, targetId uint32) error {
	p.l.Debugf("Creating buddy [%d] invitation for [%d].", targetId, actorId)
	return producer.ProviderImpl(p.l)(p.ctx)(invite2.EnvCommandTopic)(createInviteCommandProvider(actorId, worldId, targetId))
}

func (p *ProcessorImpl) Reject(actorId uint32, worldId byte, originatorId uint32) error {
	p.l.Debugf("Rejecting buddy [%d] invitation for [%d].", originatorId, actorId)
	return producer.ProviderImpl(p.l)(p.ctx)(invite2.EnvCommandTopic)(rejectInviteCommandProvider(actorId, worldId, originatorId))
}
