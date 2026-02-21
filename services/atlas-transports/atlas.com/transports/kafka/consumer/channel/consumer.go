package channel

import (
	"atlas-transports/channel"
	consumer2 "atlas-transports/kafka/consumer"
	channel2 "atlas-transports/kafka/message/channel"
	"context"

	channel3 "github.com/Chronicle20/atlas-constants/channel"
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

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(channel2.EnvEventTopicStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEventStatus))); err != nil {
			return err
		}
		return nil
	}
}

func handleEventStatus(l logrus.FieldLogger, ctx context.Context, e channel2.StatusEvent) {
	ch := channel3.NewModel(e.WorldId, e.ChannelId)
	switch e.Type {
	case channel3.StatusTypeStarted:
		l.Debugf("Registering channel [%d] for world [%d].", e.ChannelId, e.WorldId)
		_ = channel.NewProcessor(l, ctx).Register(ch)
	case channel3.StatusTypeShutdown:
		l.Debugf("Unregistering channel [%d] for world [%d].", e.ChannelId, e.WorldId)
		_ = channel.NewProcessor(l, ctx).Unregister(ch)
	default:
		l.Errorf("Unhandled event status [%s].", e.Type)
	}
}
