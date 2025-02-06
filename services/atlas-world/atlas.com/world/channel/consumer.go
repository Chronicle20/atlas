package channel

import (
	consumer2 "atlas-world/kafka/consumer"
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
			rf(consumer2.NewConfig(l)("channel_status_event")(EnvEventTopicChannelStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvEventTopicChannelStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEventStatus)))
	}
}

func handleEventStatus(l logrus.FieldLogger, ctx context.Context, e channelStatusEvent) {
	if e.Type == EventChannelStatusType {
		l.Debugf("Registering channel [%d] for world [%d] at [%s:%d].", e.ChannelId, e.WorldId, e.IpAddress, e.Port)
		_, _ = Register(l)(ctx)(e.WorldId, e.ChannelId, e.IpAddress, e.Port)
	} else if e.Type == EventChannelStatusTypeShutdown {
		l.Debugf("Unregistering channel [%d] for world [%d] at [%s:%d].", e.ChannelId, e.WorldId, e.IpAddress, e.Port)
		_ = Unregister(l)(ctx)(e.WorldId, e.ChannelId)
	} else {
		l.Errorf("Unhandled event status [%s].", e.Type)
	}
}
