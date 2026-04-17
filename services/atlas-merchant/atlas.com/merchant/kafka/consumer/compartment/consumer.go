package compartment

import (
	consumer2 "atlas-merchant/kafka/consumer"
	"atlas-merchant/kafka/message/compartment"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("compartment_status")(compartment.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		t, _ := topic.EnvProvider(l)(compartment.EnvEventTopicStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptedEvent())))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleReleasedEvent())))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleErrorEvent())))
	}
}

func handleAcceptedEvent() message.Handler[compartment.StatusEvent[compartment.AcceptedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.AcceptedEventBody]) {
		if e.Type != compartment.StatusEventTypeAccepted {
			return
		}
		l.Debugf("Compartment accepted confirmation for transaction [%s], character [%d].", e.Body.TransactionId, e.CharacterId)
	}
}

func handleReleasedEvent() message.Handler[compartment.StatusEvent[compartment.ReleasedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.ReleasedEventBody]) {
		if e.Type != compartment.StatusEventTypeReleased {
			return
		}
		l.Debugf("Compartment released confirmation for transaction [%s], character [%d].", e.Body.TransactionId, e.CharacterId)
	}
}

func handleErrorEvent() message.Handler[compartment.StatusEvent[compartment.ErrorEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.ErrorEventBody]) {
		if e.Type != compartment.StatusEventTypeError {
			return
		}
		l.Errorf("Compartment error [%s] for transaction [%s], character [%d].", e.Body.ErrorCode, e.Body.TransactionId, e.CharacterId)
	}
}
