package asset

import (
	"atlas-inventory/compartment"
	consumer2 "atlas-inventory/kafka/consumer"
	"atlas-inventory/kafka/message/asset"
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
			rf(consumer2.NewConfig(l)("asset_command")(asset.EnvCommandTopicAssetExpire)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(asset.EnvCommandTopicAssetExpire)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleExpireCommand(db))))
		}
	}
}

func handleExpireCommand(db *gorm.DB) message.Handler[asset.ExpireCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c asset.ExpireCommand) {
		l.Debugf("Received EXPIRE command for character [%d], asset [%d], template [%d], slot [%d].",
			c.CharacterId, c.AssetId, c.TemplateId, c.Slot)

		// Only handle INVENTORY source commands
		if c.Source != "INVENTORY" {
			l.Debugf("Ignoring EXPIRE command with source [%s], expected INVENTORY.", c.Source)
			return
		}

		// Determine if this is a cash item based on inventory type
		isCash := inventory.Type(c.InventoryType) == inventory.TypeValueCash

		err := compartment.NewProcessor(l, ctx, db).ExpireAssetAndEmit(
			c.TransactionId,
			c.CharacterId,
			inventory.Type(c.InventoryType),
			c.Slot,
			isCash,
			c.ReplaceItemId,
			c.ReplaceMessage,
		)
		if err != nil {
			l.WithError(err).Errorf("Failed to expire asset [%d] for character [%d].", c.AssetId, c.CharacterId)
		}
	}
}
