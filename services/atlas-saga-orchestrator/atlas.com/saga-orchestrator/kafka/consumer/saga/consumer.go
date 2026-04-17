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

// extractInboundCharacterCreationIds mirrors saga2.ExtractCharacterCreationIds
// but reads directly from the inbound command (pre-cache-insert). Needed for
// the Put()-error path where the saga never entered the cache.
func extractInboundCharacterCreationIds(c saga2.Saga) (accountId, characterId uint32) {
	for _, step := range c.Steps() {
		if step.Action() != saga2.CreateCharacter {
			continue
		}
		if p, ok := step.Payload().(saga2.CharacterCreatePayload); ok {
			accountId = p.AccountId
		}
		return
	}
	return
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("saga_command")(saga.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(saga.EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSagaCommand))); err != nil {
			return err
		}
		return nil
	}
}

func handleSagaCommand(l logrus.FieldLogger, ctx context.Context, c saga2.Saga) {
	logger := l.WithFields(logrus.Fields{
		"transaction_id": c.TransactionId().String(),
		"saga_type":      c.SagaType(),
		"initiated_by":   c.InitiatedBy(),
		"steps_count":    c.StepCount(),
	})

	logger.Info("Handling saga command")

	processor := saga2.NewProcessor(logger, ctx)
	err := processor.Put(c)
	if err != nil {
		logger.WithError(err).Error("Failed to insert saga into cache")

		// Emit a Failed event so downstream waiters (the factory bridge, the
		// login seed consumer) do not hang forever on a saga that never
		// actually entered step execution. See PRD §4.2 / plan Phase 3.1.
		//
		// The terminal-state guard in the cache does not apply here because
		// processor.Put() failed to install the entry. A duplicate emission
		// is not possible at this site: if Put() succeeded, we'd take the
		// success branch; if it failed, no timer or StepCompleted exists to
		// race with.
		accountId, characterId := extractInboundCharacterCreationIds(c)
		emitErr := saga2.EmitSagaFailedByIds(logger, ctx,
			c.TransactionId(), string(c.SagaType()), accountId, characterId,
			saga.ErrorCodeUnknown, err.Error(), "")
		if emitErr != nil {
			logger.WithError(emitErr).Error("Failed to emit saga failed event on Put() error")
		}
		return
	}

	logger.Info("Successfully inserted saga into cache")
}
