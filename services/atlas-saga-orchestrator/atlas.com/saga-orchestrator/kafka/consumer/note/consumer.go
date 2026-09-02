package note

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	note2 "atlas-saga-orchestrator/kafka/message/note"
	"atlas-saga-orchestrator/saga"
	"context"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("note_status_event")(note2.EnvEventTopicNoteStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		var err error
		t, err = topic.EnvProvider(l)(note2.EnvEventTopicNoteStatus)()
		if err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreatedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateFailedEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleCreatedEvent(l logrus.FieldLogger, ctx context.Context, e note2.StatusEvent[note2.StatusEventCreatedBody]) {
	if e.Type != note2.StatusEventTypeCreated {
		return
	}

	// Skip events without a transaction id (REST-created / non-saga notes).
	if e.TransactionId == uuid.Nil {
		return
	}

	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindNoteCreated); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"note_id":        e.Body.NoteId,
		"receiver_id":    e.CharacterId,
	}).Debug("Note created, marking saga step as completed.")

	_ = p.StepCompleted(e.TransactionId, true)
}

func handleCreateFailedEvent(l logrus.FieldLogger, ctx context.Context, e note2.StatusEvent[note2.StatusEventCreateFailedBody]) {
	if e.Type != note2.StatusEventTypeCreateFailed {
		return
	}

	if e.TransactionId == uuid.Nil {
		return
	}

	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindNoteCreateFailed); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"receiver_id":    e.CharacterId,
		"reason":         e.Body.Reason,
	}).Warn("Note creation failed, failing saga step.")

	_ = p.StepCompleted(e.TransactionId, false)
}
