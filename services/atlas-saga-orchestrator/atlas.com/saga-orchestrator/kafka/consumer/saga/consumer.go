package saga

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	"atlas-saga-orchestrator/kafka/message/saga"
	saga2 "atlas-saga-orchestrator/saga"
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
			rf(consumer2.NewConfig(l)("saga_command")(saga.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(saga.EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSagaCommand)))
	}
}

func handleSagaCommand(l logrus.FieldLogger, ctx context.Context, c saga2.Saga) {
	logger := l.WithFields(logrus.Fields{
		"transaction_id": c.TransactionId.String(),
		"saga_type":      c.SagaType,
		"initiated_by":   c.InitiatedBy,
		"steps_count":    len(c.Steps),
	})

	logger.Info("Handling saga command")

	processor := saga2.NewProcessor(logger, ctx)
	err := processor.Put(c)
	if err != nil {
		logger.WithError(err).Error("Failed to insert saga into cache")
		return
	}

	logger.Info("Successfully inserted saga into cache")
}
