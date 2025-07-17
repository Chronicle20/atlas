package saga

import (
	consumer2 "atlas-character-factory/kafka/consumer"
	"atlas-character-factory/kafka/message/saga"
	seedMessage "atlas-character-factory/kafka/message/seed"
	"atlas-character-factory/kafka/producer/seed"
	"atlas-character-factory/kafka/producer"
	"atlas-character-factory/factory"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("saga_event")(saga.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(saga.EnvStatusEventTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSagaCompletedEvent)))
	}
}

func handleSagaCompletedEvent(l logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventCompletedBody]) {
	if e.Type != saga.StatusEventTypeCompleted {
		return
	}

	// Mark the saga as completed and check if both sagas are now complete
	tracker, bothComplete := factory.MarkSagaCompleted(e.Body.TransactionId)
	if !bothComplete {
		l.Debugf("Saga [%s] completed, but waiting for the other saga to complete", e.Body.TransactionId.String())
		return
	}

	l.Debugf("Both character creation sagas completed for account [%d] character [%d], emitting seed completion event", 
		tracker.AccountId, tracker.CharacterId)

	// Emit seed completion event
	seedEventProvider := seed.CreatedEventStatusProvider(tracker.AccountId, tracker.CharacterId)
	seedProducer := producer.ProviderImpl(l)(ctx)(seedMessage.EnvEventTopicStatus)
	err := seedProducer(seedEventProvider)
	if err != nil {
		l.WithError(err).Errorf("Failed to emit seed completion event for account [%d] character [%d]", 
			tracker.AccountId, tracker.CharacterId)
		return
	}

	l.Debugf("Seed completion event emitted successfully for account [%d] character [%d]", 
		tracker.AccountId, tracker.CharacterId)
}
