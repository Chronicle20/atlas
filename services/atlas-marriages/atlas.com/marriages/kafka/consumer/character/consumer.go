package character

import (
	"context"

	localConsumer "atlas-marriages/kafka/consumer"
	"atlas-marriages/kafka/message"
	characterMsg "atlas-marriages/kafka/message/character"
	"atlas-marriages/kafka/producer"
	marriageService "atlas-marriages/marriage"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	kafka "github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// NewConfig creates a new consumer configuration for character events
func NewConfig(l logrus.FieldLogger) func(name string) func(token string) func(groupId string) consumer.Config {
	return localConsumer.NewConfig(l)
}

// InitHandlers initializes all character event handlers
func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(characterMsg.EnvEventTopicStatus)()
			// Character deleted event handler
			if _, err := rf(t, kafka.AdaptHandler(kafka.PersistentConfig(handleCharacterDeleted(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// handleCharacterDeleted handles character deleted status events
func handleCharacterDeleted(db *gorm.DB) kafka.Handler[characterMsg.StatusEvent[characterMsg.DeletedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event characterMsg.StatusEvent[characterMsg.DeletedStatusEventBody]) {
		processor := marriageService.NewProcessor(l, ctx, db)
		l.WithFields(logrus.Fields{
			"type":        event.Type,
			"characterId": event.CharacterId,
			"worldId":     event.WorldId,
		}).Debug("Processing character deleted event")

		if event.Type != characterMsg.StatusEventTypeDeleted {
			return
		}

		transactionId := uuid.New()

		// Process the character deletion using the same business logic
		err := processor.HandleCharacterDeletionAndEmit(transactionId, event.CharacterId)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"deletedCharacterId": event.CharacterId,
				"worldId":            event.WorldId,
			}).Error("Failed to process character deletion")

			// Emit error event
			errorProvider := marriageService.MarriageErrorEventProvider(
				event.CharacterId,
				"CHARACTER_DELETION_FAILED",
				"CHARACTER_DELETION_ERROR",
				err.Error(),
				"character_deletion",
			)
			if emitErr := message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
				return buf.Put("EVENT_TOPIC_MARRIAGE_STATUS", errorProvider)
			}); emitErr != nil {
				l.WithError(emitErr).Error("Failed to emit error event for character deletion failure")
			}
			return
		}

		l.WithFields(logrus.Fields{
			"deletedCharacterId": event.CharacterId,
			"worldId":            event.WorldId,
		}).Info("Character deletion processed successfully")
	}
}

// InitConsumers initializes the character event consumers
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			// Initialize consumer for character status events
			config := NewConfig(l)("character_command")(characterMsg.EnvEventTopicStatus)(consumerGroupId)

			// Set up header parsers for tenant and span context
			rf(config,
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
			)
		}
	}
}
