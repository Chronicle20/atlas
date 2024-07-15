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

func StatusConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerNameStatus)(topicTokenStatus)(groupId)
	}
}

func handleStatus() message.Handler[channelServerEvent] {
	return func(l logrus.FieldLogger, span opentracing.Span, event channelServerEvent) {
		if event.Status == EventStatusStarted {
			l.Debugf("Registering channel [%d] for world [%d] at [%s:%d].", event.ChannelId, event.WorldId, event.IpAddress, event.Port)
			_, _ = Register(l, event.Tenant)(event.WorldId, event.ChannelId, event.IpAddress, event.Port)
		} else if event.Status == EventStatusShutdown {
			l.Debugf("Unregistering channel [%d] for world [%d] at [%s:%d].", event.ChannelId, event.WorldId, event.IpAddress, event.Port)
			_ = Unregister(l, event.Tenant)(event.WorldId, event.ChannelId)
		} else {
			l.Errorf("Unhandled event status [%s].", event.Status)
		}
	}
}

func StatusRegister(l *logrus.Logger) (string, handler.Handler) {
	return kafka.LookupTopic(l)(topicTokenStatus), message.AdaptHandler(message.PersistentConfig(handleStatus()))
}
