package invite

import (
	invite2 "atlas-buddies/kafka/message/invite"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Create(actorId uint32, worldId world.Id, targetId uint32) error {
	p.l.Debugf("Creating buddy [%d] invitation for [%d].", targetId, actorId)
	return producer.ProviderImpl(p.l)(p.ctx)(invite2.EnvCommandTopic)(createInviteCommandProvider(character.Id(actorId), worldId, character.Id(targetId)))
}

func (p *ProcessorImpl) Reject(actorId uint32, worldId world.Id, originatorId uint32) error {
	p.l.Debugf("Rejecting buddy [%d] invitation for [%d].", originatorId, actorId)
	return producer.ProviderImpl(p.l)(p.ctx)(invite2.EnvCommandTopic)(rejectInviteCommandProvider(character.Id(actorId), worldId, character.Id(originatorId)))
}
