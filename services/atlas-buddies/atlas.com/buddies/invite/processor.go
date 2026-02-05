package invite

import (
	invite2 "atlas-buddies/kafka/message/invite"
	"atlas-buddies/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Create(actorId uint32, worldId world.Id, targetId uint32) error
	Reject(actorId uint32, worldId world.Id, originatorId uint32) error
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

func (p *ProcessorImpl) Create(actorId uint32, worldId world.Id, targetId uint32) error {
	p.l.Debugf("Creating buddy [%d] invitation for [%d].", targetId, actorId)
	return producer.ProviderImpl(p.l)(p.ctx)(invite2.EnvCommandTopic)(createInviteCommandProvider(character.Id(actorId), worldId, character.Id(targetId)))
}

func (p *ProcessorImpl) Reject(actorId uint32, worldId world.Id, originatorId uint32) error {
	p.l.Debugf("Rejecting buddy [%d] invitation for [%d].", originatorId, actorId)
	return producer.ProviderImpl(p.l)(p.ctx)(invite2.EnvCommandTopic)(rejectInviteCommandProvider(character.Id(actorId), worldId, character.Id(originatorId)))
}
