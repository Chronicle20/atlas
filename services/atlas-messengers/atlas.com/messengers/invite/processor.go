package invite

import (
	"atlas-messengers/kafka/message/invite"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Processor interface {
	Create(transactionID uuid.UUID, actorId uint32, worldId world.Id, messengerId uint32, targetId uint32) error
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

func (p *ProcessorImpl) Create(transactionID uuid.UUID, actorId uint32, worldId world.Id, messengerId uint32, targetId uint32) error {
	p.l.Debugf("Creating messenger [%d] invitation for [%d] from [%d].", messengerId, targetId, actorId)
	return producer.ProviderImpl(p.l)(p.ctx)(invite.EnvCommandTopic)(createInviteCommandProvider(transactionID, actorId, messengerId, worldId, targetId))
}
