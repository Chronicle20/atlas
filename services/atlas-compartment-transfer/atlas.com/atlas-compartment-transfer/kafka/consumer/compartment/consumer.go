package compartment

import (
	consumer2 "atlas-compartment-transfer/kafka/consumer"
	"atlas-compartment-transfer/kafka/message/compartment"
	"atlas-compartment-transfer/transfer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("compartment_transfer_command")(compartment.EnvCommandTopicCompartmentTransfer)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(compartment.EnvCommandTopicCompartmentTransfer)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleTransferCommand)))
	}
}

func handleTransferCommand(l logrus.FieldLogger, ctx context.Context, e compartment.TransferCommand) {
	_ = transfer.NewProcessor(l, ctx).ProcessAndEmit(e)
}
