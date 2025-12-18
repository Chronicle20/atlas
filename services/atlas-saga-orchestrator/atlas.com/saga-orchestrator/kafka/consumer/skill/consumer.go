package skill

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	skill2 "atlas-saga-orchestrator/kafka/message/skill"
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
			rf(consumer2.NewConfig(l)("skill_status_event")(skill2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(skill2.EnvStatusEventTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSkillCreatedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSkillUpdatedEvent)))
	}
}

func handleSkillCreatedEvent(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventCreatedBody]) {
	if e.Type != skill2.StatusEventTypeCreated {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleSkillUpdatedEvent(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventUpdatedBody]) {
	if e.Type != skill2.StatusEventTypeUpdated {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}