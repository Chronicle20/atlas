package skill

import (
	"atlas-saga-orchestrator/kafka/message"
	skill2 "atlas-saga-orchestrator/kafka/message/skill"
	"atlas-saga-orchestrator/kafka/producer"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"time"
)

type Processor interface {
	RequestCreateAndEmit(transactionId uuid.UUID, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error
	RequestCreate(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error
	RequestUpdateAndEmit(transactionId uuid.UUID, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error
	RequestUpdate(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		p:   producer.ProviderImpl(l)(ctx),
	}
}

func (p *ProcessorImpl) RequestCreateAndEmit(transactionId uuid.UUID, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.RequestCreate(mb)(transactionId, characterId, skillId, level, masterLevel, expiration)
	})
}

func (p *ProcessorImpl) RequestCreate(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
	return func(transactionId uuid.UUID, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
		return mb.Put(skill2.EnvCommandTopic, RequestCreateProvider(transactionId, characterId, skillId, level, masterLevel, expiration))
	}
}

func (p *ProcessorImpl) RequestUpdateAndEmit(transactionId uuid.UUID, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.RequestUpdate(mb)(transactionId, characterId, skillId, level, masterLevel, expiration)
	})
}

func (p *ProcessorImpl) RequestUpdate(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
	return func(transactionId uuid.UUID, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
		return mb.Put(skill2.EnvCommandTopic, RequestUpdateProvider(transactionId, characterId, skillId, level, masterLevel, expiration))
	}
}
