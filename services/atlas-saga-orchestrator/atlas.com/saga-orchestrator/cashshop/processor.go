package cashshop

import (
	"atlas-saga-orchestrator/kafka/message"
	"atlas-saga-orchestrator/kafka/message/cashshop"
	"atlas-saga-orchestrator/kafka/producer"
	"context"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	AwardCurrencyAndEmit(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error
	AwardCurrency(mb *message.Buffer) func(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error
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
