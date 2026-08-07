package cashshop

import (
	"atlas-saga-orchestrator/kafka/message"
	"atlas-saga-orchestrator/kafka/message/cashshop"
	cashshopCompartment "atlas-saga-orchestrator/kafka/message/cashshop/compartment"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	AwardCurrencyAndEmit(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error
	AwardCurrency(mb *message.Buffer) func(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error
	AcceptAndEmit(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16) error
	Accept(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16) error
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) AwardCurrencyAndEmit(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.AwardCurrency(mb)(transactionId, accountId, currencyType, amount)
	})
}

func (p *ProcessorImpl) AwardCurrency(mb *message.Buffer) func(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
	return func(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
		// Wallet AdjustCurrency goes to COMMAND_TOPIC_WALLET — the topic the cashshop
		// wallet handler subscribes to. (Accept/Release below are cash-COMPARTMENT
		// ops and correctly use the compartment topic.) Publishing to the compartment
		// topic instead left the command on a consumer that ignores it, so every NX
		// debit (MTS buy/bid/settle) timed out with no wallet-updated event.
		return mb.Put(cashshop.EnvCommandTopicWallet, AdjustCurrencyProvider(transactionId, accountId, currencyType, amount))
	}
}

func (p *ProcessorImpl) AcceptAndEmit(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Accept(mb)(transactionId, characterId, accountId, compartmentId, compartmentType, cashId, templateId, quantity, commodityId, purchasedBy, flag)
	})
}

func (p *ProcessorImpl) Accept(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16) error {
	return func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16) error {
		return mb.Put(cashshopCompartment.EnvCommandTopic, AcceptCommandProvider(characterId, accountId, compartmentId, compartmentType, transactionId, cashId, templateId, quantity, commodityId, purchasedBy, flag))
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
