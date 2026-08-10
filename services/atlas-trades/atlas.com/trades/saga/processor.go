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
