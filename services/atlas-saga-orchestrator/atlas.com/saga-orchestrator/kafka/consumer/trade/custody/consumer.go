package custody

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	tradeCustody "atlas-saga-orchestrator/kafka/message/trade/custody"
	"atlas-saga-orchestrator/saga"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// InitConsumers registers the EVENT_TOPIC_TRADE_CUSTODY_STATUS consumer. It
// mirrors the MTS custody status consumer: atlas-trades emits custody acks
// (ACCEPTED / RELEASED / RESTORED / ERROR) carrying the transactionId, and the
// orchestrator feeds them into the saga step-completion path.
//
// RESTORED is deliberately NOT routed into StepCompleted. A restore is a
// compensating inverse dispatched after its saga already terminated, so there
// is no pending step for it to complete; completing one would advance whatever
// step happened to be current. It is logged and dropped, matching how the MTS
// consumer treats its own restore ack.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("trade_custody_status_event")(tradeCustody.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		var err error
		t, err = topic.EnvProvider(l)(tradeCustody.EnvStatusEventTopic)()
		if err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleReleasedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleErrorEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleAcceptedEvent(l logrus.FieldLogger, ctx context.Context, e tradeCustody.StatusEvent[tradeCustody.StatusEventAcceptedBody]) {
	if e.Type != tradeCustody.StatusEventTypeAccepted {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindTradeCustodyAccepted); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"escrow_id":      e.Body.EscrowId.String(),
	}).Debug("Trade escrow accepted successfully")

	_ = p.StepCompleted(e.TransactionId, true)
}

func handleReleasedEvent(l logrus.FieldLogger, ctx context.Context, e tradeCustody.StatusEvent[tradeCustody.StatusEventReleasedBody]) {
	if e.Type != tradeCustody.StatusEventTypeReleased {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindTradeCustodyReleased); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"escrow_id":      e.Body.EscrowId.String(),
	}).Debug("Trade escrow released successfully")

	_ = p.StepCompleted(e.TransactionId, true)
}

func handleErrorEvent(l logrus.FieldLogger, ctx context.Context, e tradeCustody.StatusEvent[tradeCustody.StatusEventErrorBody]) {
	if e.Type != tradeCustody.StatusEventTypeError {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindTradeCustodyError); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"escrow_id":      e.Body.EscrowId.String(),
		"error":          e.Body.Error,
	}).Error("Trade custody operation failed")

	_ = p.StepCompleted(e.TransactionId, false)
}
