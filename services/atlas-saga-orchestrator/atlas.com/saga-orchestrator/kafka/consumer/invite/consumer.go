package invite

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	"atlas-saga-orchestrator/kafka/message/invite"
	"atlas-saga-orchestrator/saga"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("invite_status_event")(invite.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(invite.EnvEventStatusTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreatedStatusEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptedStatusEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRejectedStatusEvent)))
	}
}

func handleCreatedStatusEvent(l logrus.FieldLogger, ctx context.Context, e invite.StatusEvent[invite.CreatedEventBody]) {
	if e.Type != invite.EventInviteStatusTypeCreated {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"invite_type":    e.InviteType,
		"reference_id":   e.ReferenceId,
		"originator_id":  e.Body.OriginatorId,
		"target_id":      e.Body.TargetId,
	}).Debug("Received invite created event.")

	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleAcceptedStatusEvent(l logrus.FieldLogger, ctx context.Context, e invite.StatusEvent[invite.AcceptedEventBody]) {
	if e.Type != invite.EventInviteStatusTypeAccepted {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"invite_type":    e.InviteType,
		"reference_id":   e.ReferenceId,
		"originator_id":  e.Body.OriginatorId,
		"target_id":      e.Body.TargetId,
	}).Debug("Received invite accepted event.")

	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleRejectedStatusEvent(l logrus.FieldLogger, ctx context.Context, e invite.StatusEvent[invite.RejectedEventBody]) {
	if e.Type != invite.EventInviteStatusTypeRejected {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"invite_type":    e.InviteType,
		"reference_id":   e.ReferenceId,
		"originator_id":  e.Body.OriginatorId,
		"target_id":      e.Body.TargetId,
	}).Debug("Received invite rejected event.")

	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, false)
}
