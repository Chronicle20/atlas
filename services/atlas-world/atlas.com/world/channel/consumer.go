package channel

import (
	"atlas-world/kafka"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const (
	consumerNameStatus = "channel_service_event"
)

func EventStatusConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerNameStatus)(EnvEventTopicChannelStatus)(groupId)
	}
}

func handleEventStatus() message.Handler[channelStatusEvent] {
	return func(l logrus.FieldLogger, span opentracing.Span, event channelStatusEvent) {
		if event.Type == EventChannelStatusType {
			l.Debugf("Registering channel [%d] for world [%d] at [%s:%d].", event.ChannelId, event.WorldId, event.IpAddress, event.Port)
			_, _ = Register(l, event.Tenant)(event.WorldId, event.ChannelId, event.IpAddress, event.Port)
		} else if event.Type == EventChannelStatusTypeShutdown {
			l.Debugf("Unregistering channel [%d] for world [%d] at [%s:%d].", event.ChannelId, event.WorldId, event.IpAddress, event.Port)
			_ = Unregister(l, event.Tenant)(event.WorldId, event.ChannelId)
		} else {
			l.Errorf("Unhandled event status [%s].", event.Type)
		}
	}
}

func EventStatusRegister(l *logrus.Logger) (string, handler.Handler) {
	return kafka.LookupTopic(l)(EnvEventTopicChannelStatus), message.AdaptHandler(message.PersistentConfig(handleEventStatus()))
}
