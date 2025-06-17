package asset

import (
	consumer2 "atlas-pets/kafka/consumer"
	"atlas-pets/kafka/message/asset"
	"atlas-pets/pet"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
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
			rf(consumer2.NewConfig(l)("asset_status_event")(asset.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(asset.EnvEventTopicStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetDeleted(db))))
		}
	}
}

func handleAssetDeleted(db *gorm.DB) message.Handler[asset.StatusEvent[asset.CreatedStatusEventBody[any]]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.CreatedStatusEventBody[any]]) {
		if e.Type != asset.StatusEventTypeDeleted {
			return
		}

		it, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
		if !ok {
			return
		}

		if it != inventory.TypeValueCash {
			return
		}

		if item.GetClassification(item.Id(e.TemplateId)) != item.ClassificationPet {
			return
		}

		_ = pet.NewProcessor(l, ctx, db).DeleteOnRemoveAndEmit(e.CharacterId, e.TemplateId, e.Slot)
	}
}
