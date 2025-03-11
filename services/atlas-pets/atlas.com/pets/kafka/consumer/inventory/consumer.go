package inventory

import (
	consumer2 "atlas-pets/kafka/consumer"
	"atlas-pets/pet"
	"context"
	"github.com/Chronicle20/atlas-constants/item"
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
			rf(consumer2.NewConfig(l)("inventory_changed_event")(EnvEventInventoryChanged)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvEventInventoryChanged)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryAdd(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryDelete(db))))
		}
	}
}

func handleInventoryAdd(db *gorm.DB) message.Handler[inventoryChangedEvent[inventoryChangedItemAddBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventoryChangedEvent[inventoryChangedItemAddBody]) {
		if e.Type != ChangedTypeAdd {
			return
		}

		if item.GetClassification(item.Id(e.Body.ItemId)) != item.Classification(500) {
			return
		}

		_ = pet.CreateOnAward(l)(ctx)(db)(e.CharacterId, e.Body.ItemId, e.Slot)
	}
}

func handleInventoryDelete(db *gorm.DB) message.Handler[inventoryChangedEvent[inventoryChangedItemRemoveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventoryChangedEvent[inventoryChangedItemRemoveBody]) {
		if e.Type != ChangedTypeRemove {
			return
		}

		if item.GetClassification(item.Id(e.Body.ItemId)) != item.Classification(500) {
			return
		}

		_ = pet.DeleteOnRemove(l)(ctx)(db)(e.CharacterId, e.Body.ItemId, e.Slot)
	}
}
