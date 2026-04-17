package saga

import (
	consumer2 "atlas-character-factory/kafka/consumer"
	"atlas-character-factory/kafka/message/saga"
	seedMessage "atlas-character-factory/kafka/message/seed"
	"atlas-character-factory/kafka/producer"
	"atlas-character-factory/kafka/producer/seed"
	"context"

	sharedsaga "github.com/Chronicle20/atlas-saga"

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

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(saga.EnvStatusEventTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSagaCompletedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSagaFailedEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleSagaCompletedEvent(l logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventCompletedBody]) {
	if e.Type != saga.StatusEventTypeCompleted {
		return
	}

	// Only handle CharacterCreation saga completions
	if e.Body.SagaType != string(sharedsaga.CharacterCreation) {
		return
	}

	accountId := extractUint32(e.Body.Results, "accountId")
	characterId := extractUint32(e.Body.Results, "characterId")

	if accountId == 0 || characterId == 0 {
		l.WithField("transaction_id", e.TransactionId.String()).
			Warn("CharacterCreation saga completed but missing accountId or characterId in results")
		return
	}

	l.Debugf("CharacterCreation saga [%s] completed for account [%d] character [%d], emitting seed completion event",
		e.TransactionId.String(), accountId, characterId)

	seedEventProvider := seed.CreatedEventStatusProvider(accountId, characterId)
	seedProducer := producer.ProviderImpl(l)(ctx)(seedMessage.EnvEventTopicStatus)
	err := seedProducer(seedEventProvider)
	if err != nil {
		l.WithError(err).Errorf("Failed to emit seed completion event for account [%d] character [%d]",
			accountId, characterId)
		return
	}

	l.Debugf("Seed completion event emitted successfully for account [%d] character [%d]",
		accountId, characterId)
}

// handleSagaFailedEvent re-emits a CharacterCreation saga failure as a FAILED
// event on EVENT_TOPIC_SEED_STATUS so atlas-login can write
// AddCharacterCodeUnknownError to the waiting session (PRD §4.4 / plan Phase 7).
//
// Filtered strictly by sagaType so other saga types' failures (inventory,
// storage, gachapon, etc.) do NOT leak onto the seed topic. No in-flight
// tracking map is needed — sagaType is the authoritative filter.
func handleSagaFailedEvent(l logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventFailedBody]) {
	if e.Type != saga.StatusEventTypeFailed {
		return
	}
	if e.Body.SagaType != string(sharedsaga.CharacterCreation) {
		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"saga_type":      e.Body.SagaType,
			"failed_step":    e.Body.FailedStep,
		}).Debug("Saga FAILED event for non-character-creation saga; dropping.")
		return
	}

	accountId := e.Body.AccountId
	if accountId == 0 {
		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"failed_step":    e.Body.FailedStep,
			"error_code":     e.Body.ErrorCode,
		}).Warn("CharacterCreation saga FAILED but accountId is 0; cannot route to login.")
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"account_id":     accountId,
		"character_id":   e.Body.CharacterId,
		"failed_step":    e.Body.FailedStep,
		"error_code":     e.Body.ErrorCode,
		"reason":         e.Body.Reason,
	}).Info("Re-emitting character-creation FAILED as seed FAILED for login handoff.")

	seedProducer := producer.ProviderImpl(l)(ctx)(seedMessage.EnvEventTopicStatus)
	if err := seedProducer(seed.FailedEventStatusProvider(accountId, e.Body.Reason)); err != nil {
		l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"account_id":     accountId,
		}).Error("Failed to emit seed FAILED event.")
	}
}

func extractUint32(m map[string]any, key string) uint32 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case uint32:
		return val
	case float64:
		return uint32(val)
	case int:
		return uint32(val)
	case int64:
		return uint32(val)
	default:
		return 0
	}
}
