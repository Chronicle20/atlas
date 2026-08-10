// Package saga consumes EVENT_TOPIC_SAGA_STATUS, the orchestrator's terminal
// saga outcomes, and routes the ones belonging to a settlement this service
// submitted into the trade processor.
//
// The topic carries EVERY saga in the deployment, so both handlers discriminate
// on StatusEvent.Type before touching Body. The FILTER is then the durable
// settlement record: a status whose transaction id matches no unresolved
// settlement is another service's saga, or a redelivery of one already handled,
// and the processor drops it. Filtering on the in-memory ROOM instead would
// drop the very case the record exists for — a terminal status redelivered
// after a restart, when the trade has executed but the room is gone.
//
// The FAILED body's characterId is deliberately NOT used to identify a
// participant. It names the failed EXPANDED step's character — for a trade
// settlement, whichever side's release, accept or meso award broke — which is
// not a role. Both participants come from the room the transaction id resolves.
package saga

import (
	consumer2 "atlas-trades/kafka/consumer"
	sagamsg "atlas-trades/kafka/message/saga"
	"atlas-trades/trade"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("saga_status_event")(sagamsg.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(sagamsg.EnvStatusEventTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSagaCompleted(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSagaFailed(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// handleSagaCompleted resolves whichever of atlas-trades' three sagas this
// status belongs to.
//
// One topic carries every saga in the deployment, and atlas-trades now submits
// three kinds — an item stage, a meso stake, and a settlement — none of which
// the envelope distinguishes: it carries a transaction id and nothing else. So
// the routing is by OWNERSHIP, asked in ascending cost: a stage owns an escrow
// row with its id, a stake owns a pending meso row with its id, and a settlement
// owns a durable record with its id. Each probe reports whether it claimed the
// transaction, and the first claim wins.
//
// Order matters only for cost, not correctness — the three id spaces are
// disjoint, because each id is minted fresh for exactly one saga.
//
// For a settlement this is the ONLY place SETTLED originates (design §6.4): the
// client renders its "received %d mesos after fees" line from its own character
// data, so the meso award has to have landed before the dialog closes.
func handleSagaCompleted(db *gorm.DB) message.Handler[sagamsg.StatusEvent[sagamsg.StatusEventCompletedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e sagamsg.StatusEvent[sagamsg.StatusEventCompletedBody]) {
		if e.Type != sagamsg.StatusEventTypeCompleted {
			return
		}
		p := trade.NewProcessor(l, ctx, db)

		claimed, err := p.StageSucceeded(uuid.New(), e.TransactionId)
		if err != nil {
			l.WithError(err).Errorf("Unable to confirm the trade stage of transaction [%s].", e.TransactionId.String())
			return
		}
		if claimed {
			return
		}

		claimed, err = p.MesoStageSucceeded(uuid.New(), e.TransactionId)
		if err != nil {
			l.WithError(err).Errorf("Unable to commit the trade meso stake of transaction [%s].", e.TransactionId.String())
			return
		}
		if claimed {
			return
		}

		if err := p.SettlementSucceeded(uuid.New(), e.TransactionId); err != nil {
			l.WithError(err).Errorf("Unable to record the settlement of transaction [%s].", e.TransactionId.String())
		}
	}
}

// handleSagaFailed is handleSagaCompleted's twin: same ownership routing, the
// failing outcome of each. A failed settlement closes both dialogs with LEAVE 8
// (design §3.3); a failed stage or stake refuses only the acting player, because
// the counterparty was never told either existed.
func handleSagaFailed(db *gorm.DB) message.Handler[sagamsg.StatusEvent[sagamsg.StatusEventFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e sagamsg.StatusEvent[sagamsg.StatusEventFailedBody]) {
		if e.Type != sagamsg.StatusEventTypeFailed {
			return
		}
		reason := e.Body.Reason
		if reason == "" {
			reason = e.Body.ErrorCode
		}
		p := trade.NewProcessor(l, ctx, db)

		claimed, err := p.StageFailed(uuid.New(), e.TransactionId, reason)
		if err != nil {
			l.WithError(err).Errorf("Unable to refuse the trade stage of transaction [%s].", e.TransactionId.String())
			return
		}
		if claimed {
			return
		}

		claimed, err = p.MesoStageFailed(uuid.New(), e.TransactionId, reason)
		if err != nil {
			l.WithError(err).Errorf("Unable to abandon the trade meso stake of transaction [%s].", e.TransactionId.String())
			return
		}
		if claimed {
			return
		}

		if err := p.SettlementFailed(uuid.New(), e.TransactionId, reason); err != nil {
			l.WithError(err).Errorf("Unable to close the trade of transaction [%s] after its settlement failed.", e.TransactionId.String())
		}
	}
}
