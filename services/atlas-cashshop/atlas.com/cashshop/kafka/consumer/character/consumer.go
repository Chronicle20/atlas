package character

import (
	consumer2 "atlas-cashshop/kafka/consumer"
	"atlas-cashshop/kafka/consumer/character/compartment"
	"atlas-cashshop/kafka/message/character"
	"atlas-cashshop/wishlist"
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
			rf(consumer2.NewConfig(l)("character_status_event")(character.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			compartment.InitConsumers(l)(rf)(consumerGroupId)
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(character.EnvEventTopicStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted(db))))

			// Register compartment handlers
			compartment.InitHandlers(l)(db)(rf)
		}
	}
}

func handleStatusEventDeleted(db *gorm.DB) message.Handler[character.StatusEvent[character.DeletedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character.StatusEvent[character.DeletedStatusEventBody]) {
		if e.Type != character.StatusEventTypeDeleted {
			return
		}
		_ = wishlist.NewProcessor(l, ctx, db).DeleteAllAndEmit(e.CharacterId)
	}
}
