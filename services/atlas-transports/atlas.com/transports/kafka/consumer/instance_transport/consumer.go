package instance_transport

import (
	consumer2 "atlas-transports/kafka/consumer"
	"atlas-transports/instance"
	it "atlas-transports/kafka/message/instance_transport"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
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
			rf(consumer2.NewConfig(l)("instance_transport_command")(it.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(it.EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStartCommand))); err != nil {
			return err
		}
		return nil
	}
}

func handleStartCommand(l logrus.FieldLogger, ctx context.Context, e it.Command[it.StartCommandBody]) {
	if e.Type != it.CommandStart {
		return
	}

	l.Debugf("Received instance transport start command for character [%d] route [%s].", e.CharacterId, e.Body.RouteId)

	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, 0).Build()
	err := instance.NewProcessor(l, ctx).StartTransportAndEmit(e.CharacterId, e.Body.RouteId, f)
	if err != nil {
		l.WithError(err).Errorf("Failed to start instance transport for character [%d].", e.CharacterId)
	}
}
