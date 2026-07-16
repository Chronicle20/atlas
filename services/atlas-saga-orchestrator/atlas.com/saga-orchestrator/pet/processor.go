package pet

import (
	"atlas-saga-orchestrator/kafka/message"
	pet2 "atlas-saga-orchestrator/kafka/message/pet"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GainClosenessAndEmit(transactionId uuid.UUID, petId uint32, amount uint16) error
	GainCloseness(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, amount uint16) error
	EvolveAndEmit(transactionId uuid.UUID, petId uint32) error
	Evolve(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32) error
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GainClosenessAndEmit(transactionId uuid.UUID, petId uint32, amount uint16) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.GainCloseness(mb)(transactionId, petId, amount)
	})
}

func (p *ProcessorImpl) GainCloseness(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, amount uint16) error {
	return func(transactionId uuid.UUID, petId uint32, amount uint16) error {
		return mb.Put(pet2.EnvCommandTopic, AwardClosenessProvider(transactionId, petId, amount))
	}
}

func (p *ProcessorImpl) EvolveAndEmit(transactionId uuid.UUID, petId uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Evolve(mb)(transactionId, petId)
	})
}

func (p *ProcessorImpl) Evolve(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32) error {
	return func(transactionId uuid.UUID, petId uint32) error {
		return mb.Put(pet2.EnvCommandTopic, EvolveProvider(transactionId, petId))
	}
}
