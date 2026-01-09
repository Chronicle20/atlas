package compartment

import (
	consumer2 "atlas-storage/kafka/consumer"
	"atlas-storage/kafka/message/compartment"
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
			rf(consumer2.NewConfig(l)("storage_compartment_command")(compartment.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(compartment.EnvCommandTopic)()
			_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleAcceptCommand(db))))
			_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleReleaseCommand(db))))
		}
	}
}

func handleAcceptCommand(db *gorm.DB) kafkaMessage.Handler[compartment.Command[compartment.AcceptCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment.Command[compartment.AcceptCommandBody]) {
		if c.Type != compartment.CommandAccept {
			return
		}

		err := storage.NewProcessor(l, ctx, db).AcceptAndEmit(c.WorldId, c.AccountId, c.Body)
		if err != nil {
			l.WithError(err).Errorf("Unable to accept item for account [%d] world [%d] transaction [%s].", c.AccountId, c.WorldId, c.Body.TransactionId)
		}
	}
}

func handleReleaseCommand(db *gorm.DB) kafkaMessage.Handler[compartment.Command[compartment.ReleaseCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment.Command[compartment.ReleaseCommandBody]) {
		if c.Type != compartment.CommandRelease {
			return
		}

		err := storage.NewProcessor(l, ctx, db).ReleaseAndEmit(c.WorldId, c.AccountId, c.Body)
		if err != nil {
			l.WithError(err).Errorf("Unable to release asset [%d] for account [%d] world [%d] transaction [%s].", c.Body.AssetId, c.AccountId, c.WorldId, c.Body.TransactionId)
		}
	}
}
