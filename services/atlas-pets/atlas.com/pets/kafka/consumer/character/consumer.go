package character

import (
	consumer2 "atlas-pets/kafka/consumer"
	"atlas-pets/pet"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterDeleted(db))))
		}
	}
}

func handleCharacterDeleted(db *gorm.DB) message.Handler[statusEvent[statusEventDeletedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventDeletedBody]) {
		if e.Type != StatusEventTypeDeleted {
			return
		}

		_ = pet.DeleteForCharacter(l)(ctx)(db)(e.CharacterId)
	}
}
