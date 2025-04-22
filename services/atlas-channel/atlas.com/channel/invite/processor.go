package invite

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) Accept(actorId uint32, worldId world.Id, inviteType string, referenceId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(acceptInviteCommandProvider(actorId, worldId, inviteType, referenceId))
}

func (p *Processor) Reject(actorId uint32, worldId world.Id, inviteType string, originatorId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(rejectInviteCommandProvider(actorId, worldId, inviteType, originatorId))
}
