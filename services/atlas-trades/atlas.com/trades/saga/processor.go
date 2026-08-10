package saga

import (
	"atlas-trades/kafka/message"
	sagamsg "atlas-trades/kafka/message/saga"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// Processor submits settlement sagas. It takes the caller's message buffer
// rather than writing to Kafka directly, so the saga command lands in the same
// transactional-outbox batch as the trade status events and registry-backed
// state change that accompany it: the saga publishes if and only if the
// enclosing transaction commits.
type Processor interface {
	// Settle submits the one-step trade_settlement saga for transactionId.
	Settle(mb *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.TradeSettlementPayload) error

	// Stage submits the transfer_to_trade composite one PUT_ITEM produces.
	// transactionId is the escrow row id (see BuildStage).
	Stage(mb *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.TransferToTradePayload) error

	// StageMeso submits the award_mesos that moves a meso stake into or out of
	// escrow. amount is signed: negative debits the staking player.
	StageMeso(mb *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.AwardMesosPayload) error

	// Unwind submits the trade_unwind composite a teardown produces.
	Unwind(mb *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.TradeUnwindPayload) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Settle(mb *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.TradeSettlementPayload) error {
	return func(transactionId uuid.UUID, payload sharedsaga.TradeSettlementPayload) error {
		p.l.Infof("Submitting trade settlement saga [%s] for characters [%d] and [%d].", transactionId.String(), payload.Sides[0].CharacterId, payload.Sides[1].CharacterId)
		return mb.Put(sagamsg.EnvCommandTopic, CommandProvider(Build(transactionId, payload)))
	}
}

func (p *ProcessorImpl) Stage(mb *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.TransferToTradePayload) error {
	return func(transactionId uuid.UUID, payload sharedsaga.TransferToTradePayload) error {
		p.l.Debugf("Submitting trade staging saga [%s] for character [%d] asset [%d].", transactionId.String(), payload.CharacterId, payload.AssetId)
		return mb.Put(sagamsg.EnvCommandTopic, CommandProvider(BuildStage(transactionId, payload)))
	}
}

func (p *ProcessorImpl) StageMeso(mb *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.AwardMesosPayload) error {
	return func(transactionId uuid.UUID, payload sharedsaga.AwardMesosPayload) error {
		p.l.Debugf("Submitting trade meso staging saga [%s] for character [%d], amount [%d].", transactionId.String(), payload.CharacterId, payload.Amount)
		return mb.Put(sagamsg.EnvCommandTopic, CommandProvider(BuildStageMeso(transactionId, payload)))
	}
}

func (p *ProcessorImpl) Unwind(mb *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.TradeUnwindPayload) error {
	return func(transactionId uuid.UUID, payload sharedsaga.TradeUnwindPayload) error {
		p.l.Infof("Submitting trade unwind saga [%s]: [%d] items and [%d] meso refunds.", transactionId.String(), len(payload.Items), len(payload.Mesos))
		return mb.Put(sagamsg.EnvCommandTopic, CommandProvider(BuildUnwind(transactionId, payload)))
	}
}
