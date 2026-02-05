package guild

import (
	"context"

	"atlas-saga-orchestrator/kafka/message/guild"
	"atlas-saga-orchestrator/kafka/producer"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	RequestName(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	RequestEmblem(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	RequestDisband(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	RequestCapacityIncrease(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
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

func (p *ProcessorImpl) RequestName(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	p.l.Debugf("Requesting character [%d] input guild name for creation.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(guild.EnvCommandTopic)(RequestNameProvider(transactionId, ch, characterId))
}

func (p *ProcessorImpl) RequestEmblem(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	p.l.Debugf("Requesting character [%d] input new guild emblem.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(guild.EnvCommandTopic)(RequestEmblemProvider(transactionId, ch, characterId))
}

func (p *ProcessorImpl) RequestDisband(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to disband guild.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(guild.EnvCommandTopic)(RequestDisbandProvider(transactionId, ch, characterId))
}

func (p *ProcessorImpl) RequestCapacityIncrease(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to increase guild capacity.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(guild.EnvCommandTopic)(RequestCapacityIncreaseProvider(transactionId, ch, characterId))
}
