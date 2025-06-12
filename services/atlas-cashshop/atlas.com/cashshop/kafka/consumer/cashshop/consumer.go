package cashshop

import (
	cashshop3 "atlas-cashshop/cashshop"
	consumer2 "atlas-cashshop/kafka/consumer"
	"atlas-cashshop/kafka/message/cashshop"
	"atlas-cashshop/kafka/producer"
	cashshop2 "atlas-cashshop/kafka/producer/cashshop"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
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
			rf(consumer2.NewConfig(l)("cash_shop_command")(cashshop.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(cashshop.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestPurchase(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestInventoryIncreaseByType(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestInventoryIncreaseByItem(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestStorageIncrease(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestStorageIncreaseByItem(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestCharacterSlotIncreaseByItem(db))))
		}
	}
}

func handleCommandRequestPurchase(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestPurchaseCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestPurchaseCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestPurchase {
			return
		}
		_ = cashshop3.NewProcessor(l, ctx, db).Purchase(c.CharacterId, c.Body.Currency, c.Body.SerialNumber)
	}
}

func handleCommandRequestInventoryIncreaseByType(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestInventoryIncreaseByTypeCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestInventoryIncreaseByTypeCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestInventoryIncreaseByType {
			return
		}
		_ = cashshop3.NewProcessor(l, ctx, db).PurchaseInventoryIncreaseByTypeAndEmit(c.CharacterId, c.Body.Currency, inventory.Type(c.Body.InventoryType))
	}
}

func handleCommandRequestInventoryIncreaseByItem(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestInventoryIncreaseByItemCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestInventoryIncreaseByItemCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestInventoryIncreaseByItem {
			return
		}
		_ = cashshop3.NewProcessor(l, ctx, db).PurchaseInventoryIncreaseByItemAndEmit(c.CharacterId, c.Body.Currency, c.Body.SerialNumber)
	}
}

func handleCommandRequestStorageIncrease(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestStorageIncreaseBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestStorageIncreaseBody]) {
		if c.Type != cashshop.CommandTypeRequestStorageIncrease {
			return
		}
		_ = producer.ProviderImpl(l)(ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(c.CharacterId, "UNKNOWN_ERROR"))
	}
}

func handleCommandRequestStorageIncreaseByItem(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestCharacterSlotIncreaseByItemCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestCharacterSlotIncreaseByItemCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestStorageIncreaseByItem {
			return
		}
		_ = producer.ProviderImpl(l)(ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(c.CharacterId, "UNKNOWN_ERROR"))
	}
}

func handleCommandRequestCharacterSlotIncreaseByItem(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestCharacterSlotIncreaseByItemCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestCharacterSlotIncreaseByItemCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestCharacterSlotIncreaseByItem {
			return
		}
		_ = producer.ProviderImpl(l)(ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(c.CharacterId, "UNKNOWN_ERROR"))
	}
}
