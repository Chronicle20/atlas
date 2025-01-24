package cashshop

import (
	consumer2 "atlas-buddies/kafka/consumer"
	"atlas-buddies/list"
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
			rf(consumer2.NewConfig(l)("cash_shop_status_event")(EnvEventTopicCashShopStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvEventTopicCashShopStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterEnter(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit(db))))
		}
	}
}

func handleStatusEventCharacterEnter(db *gorm.DB) message.Handler[statusEvent[characterMovementBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[characterMovementBody]) {
		if e.Type != EventCashShopStatusTypeCharacterEnter {
			return
		}
		_ = list.UpdateShopStatus(l)(ctx)(db)(e.Body.CharacterId, e.WorldId, true)
	}
}

func handleStatusEventCharacterExit(db *gorm.DB) message.Handler[statusEvent[characterMovementBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[characterMovementBody]) {
		if e.Type != EventCashShopStatusTypeCharacterExit {
			return
		}
		_ = list.UpdateShopStatus(l)(ctx)(db)(e.Body.CharacterId, e.WorldId, false)
	}
}
