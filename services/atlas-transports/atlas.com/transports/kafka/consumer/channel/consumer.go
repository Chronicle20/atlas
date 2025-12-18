package channel

import (
	"atlas-transports/channel"
	consumer2 "atlas-transports/kafka/consumer"
	channel2 "atlas-transports/kafka/message/channel"
	"context"
	channel3 "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
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
			rf(consumer2.NewConfig(l)("channel_status_event")(channel2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(channel2.EnvEventTopicStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEventStatus)))
	}
}

func handleEventStatus(l logrus.FieldLogger, ctx context.Context, e channel2.StatusEvent) {
	if e.Type == channel2.StatusTypeStarted {
		l.Debugf("Registering channel [%d] for world [%d].", e.ChannelId, e.WorldId)
		_ = channel.NewProcessor(l, ctx).Register(world.Id(e.WorldId), channel3.Id(e.ChannelId))
	} else if e.Type == channel2.StatusTypeShutdown {
		l.Debugf("Unregistering channel [%d] for world [%d].", e.ChannelId, e.WorldId)
		_ = channel.NewProcessor(l, ctx).Unregister(world.Id(e.WorldId), channel3.Id(e.ChannelId))
	} else {
		l.Errorf("Unhandled event status [%s].", e.Type)
	}
}
