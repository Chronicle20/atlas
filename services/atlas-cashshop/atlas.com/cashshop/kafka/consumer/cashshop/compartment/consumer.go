package compartment

import (
	compartment2 "atlas-cashshop/cashshop/inventory/compartment"
	consumer2 "atlas-cashshop/kafka/consumer"
	"atlas-cashshop/kafka/message/cashshop/compartment"
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
			rf(consumer2.NewConfig(l)("cash_compartment_command")(compartment.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(compartment.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleReleaseCommand(db))))
		}
	}
}

func handleAcceptCommand(db *gorm.DB) message.Handler[compartment.Command[compartment.AcceptCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment.Command[compartment.AcceptCommandBody]) {
		if c.Type != compartment.CommandAccept {
			return
		}
		_ = compartment2.NewProcessor(l, ctx, db).AcceptAndEmit(c.AccountId, c.Body.CompartmentId, compartment2.CompartmentType(c.CompartmentType), c.Body.ReferenceId, c.Body.TransactionId)
	}
}

func handleReleaseCommand(db *gorm.DB) message.Handler[compartment.Command[compartment.ReleaseCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment.Command[compartment.ReleaseCommandBody]) {
		if c.Type != compartment.CommandRelease {
			return
		}
		_ = compartment2.NewProcessor(l, ctx, db).ReleaseAndEmit(c.AccountId, c.Body.CompartmentId, compartment2.CompartmentType(c.CompartmentType), c.Body.AssetId, c.Body.TransactionId)
	}
}
