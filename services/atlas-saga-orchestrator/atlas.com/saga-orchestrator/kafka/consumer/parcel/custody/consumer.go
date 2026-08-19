package custody

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	parcelCustody "atlas-saga-orchestrator/kafka/message/parcel/custody"
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

// InitConsumers registers the EVENT_TOPIC_PARCEL_CUSTODY_STATUS consumer. It
// mirrors the MTS custody status consumer: atlas-parcel emits custody acks
// (ACCEPTED / RELEASED / ERROR) carrying the transactionId, and the
// orchestrator feeds them into the saga step-completion path.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("parcel_custody_status_event")(parcelCustody.EnvStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(parcelCustody.EnvStatusTopic)()
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

func handleAcceptedEvent(l logrus.FieldLogger, ctx context.Context, e parcelCustody.StatusEvent[parcelCustody.StatusEventAcceptedBody]) {
	if e.Type != parcelCustody.StatusEventAccepted {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindParcelCustodyAccepted); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"parcel_id":      e.Body.ParcelId.String(),
	}).Debug("Parcel accepted into custody successfully")

	_ = p.StepCompleted(e.TransactionId, true)
}

func handleReleasedEvent(l logrus.FieldLogger, ctx context.Context, e parcelCustody.StatusEvent[parcelCustody.StatusEventReleasedBody]) {
	if e.Type != parcelCustody.StatusEventReleased {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindParcelCustodyReleased); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"parcel_id":      e.Body.ParcelId.String(),
	}).Debug("Parcel released from custody successfully")

	_ = p.StepCompleted(e.TransactionId, true)
}

func handleErrorEvent(l logrus.FieldLogger, ctx context.Context, e parcelCustody.StatusEvent[parcelCustody.StatusEventErrorBody]) {
	if e.Type != parcelCustody.StatusEventError {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindParcelCustodyError); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"error":          e.Body.Error,
	}).Error("Parcel custody operation failed")

	_ = p.StepCompleted(e.TransactionId, false)
}
