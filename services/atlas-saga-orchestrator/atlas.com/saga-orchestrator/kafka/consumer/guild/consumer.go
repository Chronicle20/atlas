package guild

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	guild2 "atlas-saga-orchestrator/kafka/message/guild"
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
			rf(consumer2.NewConfig(l)("guild_status_event")(guild2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(guild2.EnvStatusEventTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleGuildRequestAgreementEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleGuildCreatedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleGuildDisbandedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleGuildEmblemUpdatedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleGuildCapacityUpdatedEvent)))
	}
}

func handleGuildRequestAgreementEvent(l logrus.FieldLogger, ctx context.Context, e guild2.StatusEvent[guild2.StatusEventRequestAgreementBody]) {
	if e.Type != guild2.StatusEventTypeRequestAgreement {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleGuildCreatedEvent(l logrus.FieldLogger, ctx context.Context, e guild2.StatusEvent[guild2.StatusEventCreatedBody]) {
	if e.Type != guild2.StatusEventTypeCreated {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleGuildDisbandedEvent(l logrus.FieldLogger, ctx context.Context, e guild2.StatusEvent[guild2.StatusEventDisbandedBody]) {
	if e.Type != guild2.StatusEventTypeDisbanded {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleGuildEmblemUpdatedEvent(l logrus.FieldLogger, ctx context.Context, e guild2.StatusEvent[guild2.StatusEventEmblemUpdatedBody]) {
	if e.Type != guild2.StatusEventTypeEmblemUpdated {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleGuildCapacityUpdatedEvent(l logrus.FieldLogger, ctx context.Context, e guild2.StatusEvent[guild2.StatusEventCapacityUpdatedBody]) {
	if e.Type != guild2.StatusEventTypeCapacityUpdated {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}