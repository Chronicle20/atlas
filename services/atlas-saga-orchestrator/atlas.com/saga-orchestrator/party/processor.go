package party

import (
	"atlas-saga-orchestrator/kafka/message/party"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	// RequestLeave emits a LEAVE command for characterId against partyId. Used
	// by the world-transfer saga (task-227) to sever party membership ahead of
	// changing the character's world.
	RequestLeave(transactionId uuid.UUID, characterId uint32, partyId uint32) error
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

func (p *ProcessorImpl) RequestLeave(transactionId uuid.UUID, characterId uint32, partyId uint32) error {
	p.l.Debugf("Character [%d] leaving party [%d].", characterId, partyId)
	return producer.ProviderImpl(p.l)(p.ctx)(party.EnvCommandTopic)(RequestLeaveProvider(transactionId, characterId, partyId))
}
