package saga

import (
	consumer2 "atlas-map-actions/kafka/consumer"
	"atlas-map-actions/kafka/message/saga"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
		return func(groupId string) {
			rf(
				consumer2.NewConfig(l)("saga_status_event")(saga.EnvStatusEventTopic)(groupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
			)
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		t, _ := topic.EnvProvider(l)(saga.EnvStatusEventTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCompleted(l))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventFailed(l))))
	}
}

func handleStatusEventCompleted(l logrus.FieldLogger) message.Handler[saga.StatusEvent[saga.StatusEventCompletedBody]] {
	return func(logger logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventCompletedBody]) {
		if e.Type != saga.StatusEventTypeCompleted {
			return
		}
		l.Debugf("Saga [%s] completed.", e.TransactionId)
	}
}

func handleStatusEventFailed(l logrus.FieldLogger) message.Handler[saga.StatusEvent[saga.StatusEventFailedBody]] {
	return func(logger logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventFailedBody]) {
		if e.Type != saga.StatusEventTypeFailed {
			return
		}
		l.Warnf("Saga [%s] failed. Error: [%s] Reason: [%s] Step: [%s].",
			e.TransactionId, e.Body.ErrorCode, e.Body.Reason, e.Body.FailedStep)
	}
}
