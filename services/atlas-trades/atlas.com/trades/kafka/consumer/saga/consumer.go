// Package saga consumes EVENT_TOPIC_SAGA_STATUS, the orchestrator's terminal
// saga outcomes, and routes the ones belonging to a settlement this service
// submitted into the trade processor.
//
// The topic carries EVERY saga in the deployment, so both handlers discriminate
// on StatusEvent.Type before touching Body, and then filter by the settlement
// transaction id: a status whose transaction id matches no live trade room is
// another service's saga (or a redelivery of one already handled) and is
// dropped.
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

// handleSagaCompleted closes a trade whose settlement saga finished. Design §6.4
// makes this the ONLY place SETTLED originates: the client renders its
// "received %d mesos after fees" line from its own character data, so the meso
// award has to have landed before the dialog closes.
func handleSagaCompleted(db *gorm.DB) message.Handler[sagamsg.StatusEvent[sagamsg.StatusEventCompletedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e sagamsg.StatusEvent[sagamsg.StatusEventCompletedBody]) {
		if e.Type != sagamsg.StatusEventTypeCompleted {
			return
		}
		p := trade.NewProcessor(l, ctx, db)
		room, ok := p.RoomBySettlement(e.TransactionId)
		if !ok {
			return
		}
		if err := p.SettlementSucceeded(uuid.New(), room.Id()); err != nil {
			l.WithError(err).Errorf("Unable to record the settlement of trade room [%s].", room.Id().String())
		}
	}
}

// handleSagaFailed closes a trade whose settlement saga was compensated, with
// LEAVE 8 to BOTH sides (design §3.3).
func handleSagaFailed(db *gorm.DB) message.Handler[sagamsg.StatusEvent[sagamsg.StatusEventFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e sagamsg.StatusEvent[sagamsg.StatusEventFailedBody]) {
		if e.Type != sagamsg.StatusEventTypeFailed {
			return
		}
		p := trade.NewProcessor(l, ctx, db)
		room, ok := p.RoomBySettlement(e.TransactionId)
		if !ok {
			return
		}
		reason := e.Body.Reason
		if reason == "" {
			reason = e.Body.ErrorCode
		}
		if err := p.SettlementFailed(uuid.New(), room.Id(), reason); err != nil {
			l.WithError(err).Errorf("Unable to close trade room [%s] after its settlement failed.", room.Id().String())
		}
	}
}
