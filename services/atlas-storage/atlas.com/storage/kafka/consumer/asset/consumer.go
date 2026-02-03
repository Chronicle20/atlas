package asset

import (
	consumer2 "atlas-storage/kafka/consumer"
	"atlas-storage/kafka/message"
	"atlas-storage/storage"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	kafkaMessage "github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("asset_command")(message.EnvCommandTopicAssetExpire)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(message.EnvCommandTopicAssetExpire)()
			_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleExpireCommand(db))))
		}
	}
}

func handleExpireCommand(db *gorm.DB) kafkaMessage.Handler[message.ExpireCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c message.ExpireCommand) {
		l.Debugf("Received EXPIRE command for account [%d], asset [%d], template [%d], source [%s].",
			c.AccountId, c.AssetId, c.TemplateId, c.Source)

		// Only handle STORAGE source commands
		if c.Source != "STORAGE" {
			l.Debugf("Ignoring EXPIRE command with source [%s], expected STORAGE.", c.Source)
			return
		}

		// Storage items are always non-cash items (cash items are in cashshop)
		isCash := false

		err := storage.NewProcessor(l, ctx, db).ExpireAndEmit(
			c.TransactionId,
			c.WorldId,
			c.AccountId,
			c.AssetId,
			isCash,
			c.ReplaceItemId,
			c.ReplaceMessage,
		)
		if err != nil {
			l.WithError(err).Errorf("Failed to expire storage asset [%d] for account [%d].", c.AssetId, c.AccountId)
		}
	}
}
