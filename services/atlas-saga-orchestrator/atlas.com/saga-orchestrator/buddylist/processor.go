package buddylist

import (
	"atlas-saga-orchestrator/kafka/message"
	buddylist2 "atlas-saga-orchestrator/kafka/message/buddylist"
	"atlas-saga-orchestrator/kafka/producer"
	"context"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	IncreaseCapacityAndEmit(transactionId uuid.UUID, characterId uint32, worldId byte, newCapacity byte) error
	IncreaseCapacity(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId byte, newCapacity byte) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
	}
}

func (p *ProcessorImpl) IncreaseCapacityAndEmit(transactionId uuid.UUID, characterId uint32, worldId byte, newCapacity byte) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.IncreaseCapacity(mb)(transactionId, characterId, worldId, newCapacity)
	})
}

func (p *ProcessorImpl) IncreaseCapacity(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId byte, newCapacity byte) error {
	return func(transactionId uuid.UUID, characterId uint32, worldId byte, newCapacity byte) error {
		return mb.Put(buddylist2.EnvCommandTopic, IncreaseCapacityProvider(characterId, worldId, newCapacity))
	}
}
