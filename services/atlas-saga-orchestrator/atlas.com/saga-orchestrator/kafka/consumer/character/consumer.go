package character

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	character2 "atlas-saga-orchestrator/kafka/message/character"
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
			rf(consumer2.NewConfig(l)("character_status_event")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterMapChangedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterExperienceChangedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterLevelChangedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterMesoChangedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterJobChangedEvent)))
	}
}

func handleCharacterMapChangedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventMapChangedBody]) {
	if e.Type != character2.StatusEventTypeMapChanged {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleCharacterExperienceChangedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.ExperienceChangedStatusEventBody]) {
	if e.Type != character2.StatusEventTypeExperienceChanged {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleCharacterLevelChangedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.LevelChangedStatusEventBody]) {
	if e.Type != character2.StatusEventTypeLevelChanged {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleCharacterMesoChangedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.MesoChangedStatusEventBody]) {
	if e.Type != character2.StatusEventTypeMesoChanged {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleCharacterJobChangedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.JobChangedStatusEventBody]) {
	if e.Type != character2.StatusEventTypeJobChanged {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}
