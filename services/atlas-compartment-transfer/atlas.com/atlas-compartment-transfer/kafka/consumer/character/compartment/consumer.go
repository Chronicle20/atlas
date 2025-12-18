package compartment

import (
	consumer2 "atlas-compartment-transfer/kafka/consumer"
	"atlas-compartment-transfer/kafka/message/character/compartment"
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
			rf(consumer2.NewConfig(l)("compartment_status_event")(compartment.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(compartment.EnvEventTopicStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleReleasedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleErrorEvent)))
	}
}

func handleAcceptedEvent(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.AcceptedEventBody]) {
	if e.Type != compartment.StatusEventTypeAccepted {
		return
	}

	_ = transfer.NewProcessor(l, ctx).HandleAcceptedAndEmit(e.Body.TransactionId)
}

func handleReleasedEvent(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.ReleasedEventBody]) {
	if e.Type != compartment.StatusEventTypeReleased {
		return
	}

	_ = transfer.NewProcessor(l, ctx).HandleReleasedAndEmit(e.Body.TransactionId)
}

func handleErrorEvent(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.ErrorEventBody]) {
	if e.Type != compartment.StatusEventTypeError {
		return
	}

	_ = transfer.NewProcessor(l, ctx).HandleErrorAndEmit(e.Body.TransactionId)
}
