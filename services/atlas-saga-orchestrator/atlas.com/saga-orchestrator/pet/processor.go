package pet

import (
	"atlas-saga-orchestrator/kafka/message"
	pet2 "atlas-saga-orchestrator/kafka/message/pet"
	"atlas-saga-orchestrator/kafka/producer"
	"context"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GainClosenessAndEmit(transactionId uuid.UUID, petId uint32, amount uint16) error
	GainCloseness(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, amount uint16) error
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

func (p *ProcessorImpl) GainClosenessAndEmit(transactionId uuid.UUID, petId uint32, amount uint16) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.GainCloseness(mb)(transactionId, petId, amount)
	})
}

func (p *ProcessorImpl) GainCloseness(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, amount uint16) error {
	return func(transactionId uuid.UUID, petId uint32, amount uint16) error {
		return mb.Put(pet2.EnvCommandTopic, GainClosenessProvider(petId, amount))
	}
}
