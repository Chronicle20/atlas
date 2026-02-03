package asset

import (
	itemModel "atlas-cashshop/cashshop/item"
	consumer2 "atlas-cashshop/kafka/consumer"
	"atlas-cashshop/kafka/message/item"
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
			rf(consumer2.NewConfig(l)("asset_command")(item.EnvCommandTopicAssetExpire)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(item.EnvCommandTopicAssetExpire)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleExpireCommand(db))))
		}
	}
}

func handleExpireCommand(db *gorm.DB) message.Handler[item.ExpireCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c item.ExpireCommand) {
		l.Debugf("Received EXPIRE command for account [%d], asset [%d], template [%d], source [%s].",
			c.AccountId, c.AssetId, c.TemplateId, c.Source)

		// Only handle CASHSHOP source commands
		if c.Source != "CASHSHOP" {
			l.Debugf("Ignoring EXPIRE command with source [%s], expected CASHSHOP.", c.Source)
			return
		}

		err := itemModel.NewProcessor(l, ctx, db).ExpireAndEmit(
			c.AssetId,
			c.ReplaceItemId,
			c.ReplaceMessage,
		)
		if err != nil {
			l.WithError(err).Errorf("Failed to expire cashshop item [%d] for account [%d].", c.AssetId, c.AccountId)
		}
	}
}
