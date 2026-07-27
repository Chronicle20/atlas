package custody

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	mtsCustody "atlas-saga-orchestrator/kafka/message/mts/custody"
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

// InitConsumers registers the EVENT_TOPIC_MTS_CUSTODY_STATUS consumer. It
// mirrors the cash-compartment status consumer: atlas-mts emits custody acks
// (ACCEPTED / RELEASED / MOVED / ERROR) carrying the transactionId, and the
// orchestrator feeds them into the saga step-completion path.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("mts_custody_status_event")(mtsCustody.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(mtsCustody.EnvStatusEventTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleReleasedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMovedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleErrorEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleAcceptedEvent(l logrus.FieldLogger, ctx context.Context, e mtsCustody.StatusEvent[mtsCustody.StatusEventAcceptedBody]) {
	if e.Type != mtsCustody.StatusEventTypeAccepted {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindMtsCustodyAccepted); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"listing_id":     e.Body.ListingId.String(),
	}).Debug("MTS listing accepted successfully")

	_ = p.StepCompleted(e.TransactionId, true)
}

func handleReleasedEvent(l logrus.FieldLogger, ctx context.Context, e mtsCustody.StatusEvent[mtsCustody.StatusEventReleasedBody]) {
	if e.Type != mtsCustody.StatusEventTypeReleased {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindMtsCustodyReleased); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"holding_id":     e.Body.HoldingId.String(),
	}).Debug("MTS holding released successfully")

	_ = p.StepCompleted(e.TransactionId, true)
}

func handleMovedEvent(l logrus.FieldLogger, ctx context.Context, e mtsCustody.StatusEvent[mtsCustody.StatusEventMovedBody]) {
	if e.Type != mtsCustody.StatusEventTypeMoved {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindMtsCustodyMoved); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"listing_id":     e.Body.ListingId.String(),
		"holding_id":     e.Body.HoldingId.String(),
	}).Debug("MTS listing moved to buyer holding successfully")

	_ = p.StepCompleted(e.TransactionId, true)
}

func handleErrorEvent(l logrus.FieldLogger, ctx context.Context, e mtsCustody.StatusEvent[mtsCustody.StatusEventErrorBody]) {
	if e.Type != mtsCustody.StatusEventTypeError {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindMtsCustodyError); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"error":          e.Body.Error,
	}).Error("MTS custody operation failed")

	_ = p.StepCompleted(e.TransactionId, false)
}
