package cashshop

import (
	"atlas-saga-orchestrator/kafka/message"
	"atlas-saga-orchestrator/kafka/message/cashshop"
	cashshopCompartment "atlas-saga-orchestrator/kafka/message/cashshop/compartment"
	"atlas-saga-orchestrator/kafka/producer"
	"context"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	AwardCurrencyAndEmit(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error
	AwardCurrency(mb *message.Buffer) func(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error
	AcceptAndEmit(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, referenceId uint32, referenceType string, referenceData []byte) error
	Accept(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, referenceId uint32, referenceType string, referenceData []byte) error
	ReleaseAndEmit(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, assetId uint32, cashId int64, templateId uint32) error
	Release(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, assetId uint32, cashId int64, templateId uint32) error
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

func (p *ProcessorImpl) AwardCurrencyAndEmit(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.AwardCurrency(mb)(transactionId, accountId, currencyType, amount)
	})
}

func (p *ProcessorImpl) AwardCurrency(mb *message.Buffer) func(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
	return func(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
		return mb.Put(cashshop.EnvCommandTopicWallet, AdjustCurrencyProvider(transactionId, accountId, currencyType, amount))
	}
}

func (p *ProcessorImpl) AcceptAndEmit(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, referenceId uint32, referenceType string, referenceData []byte) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Accept(mb)(transactionId, characterId, accountId, compartmentId, compartmentType, cashId, templateId, referenceId, referenceType, referenceData)
	})
}

func (p *ProcessorImpl) Accept(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, referenceId uint32, referenceType string, referenceData []byte) error {
	return func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, referenceId uint32, referenceType string, referenceData []byte) error {
		return mb.Put(cashshopCompartment.EnvCommandTopic, AcceptCommandProvider(characterId, accountId, compartmentId, compartmentType, transactionId, cashId, templateId, referenceId, referenceType, referenceData))
	}
}

func (p *ProcessorImpl) ReleaseAndEmit(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, assetId uint32, cashId int64, templateId uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Release(mb)(transactionId, characterId, accountId, compartmentId, compartmentType, assetId, cashId, templateId)
	})
}

func (p *ProcessorImpl) Release(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, assetId uint32, cashId int64, templateId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, assetId uint32, cashId int64, templateId uint32) error {
		return mb.Put(cashshopCompartment.EnvCommandTopic, ReleaseCommandProvider(characterId, accountId, compartmentId, compartmentType, transactionId, assetId, cashId, templateId))
	}
}
