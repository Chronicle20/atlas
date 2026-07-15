package invite

import (
	invite2 "atlas-channel/kafka/message/invite"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Accept(actorId uint32, worldId world.Id, inviteType string, referenceId uint32) error
	Reject(actorId uint32, worldId world.Id, inviteType string, originatorId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Accept(actorId uint32, worldId world.Id, inviteType string, referenceId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(invite2.EnvCommandTopic)(AcceptInviteCommandProvider(actorId, worldId, inviteType, referenceId))
}

func (p *ProcessorImpl) Reject(actorId uint32, worldId world.Id, inviteType string, originatorId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(invite2.EnvCommandTopic)(RejectInviteCommandProvider(actorId, worldId, inviteType, originatorId))
}
