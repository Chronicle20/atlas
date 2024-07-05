package channel

import (
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const (
	consumerNameStatus = "channel_service_event"
)

func StatusConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	t := lookupTopic(l)(topicTokenStatus)
	return func(groupId string) consumer.Config {
		return consumer.NewConfig[channelServerEvent](consumerNameStatus, t, groupId, handleStatus())
	}
}

func handleStatus() consumer.HandlerFunc[channelServerEvent] {
	return func(l logrus.FieldLogger, span opentracing.Span, event channelServerEvent) {
		if event.Status == EventStatusStarted {
			l.Debugf("Registering channel [%d] for world [%d] at [%s:%d].", event.ChannelId, event.WorldId, event.IpAddress, event.Port)
			GetChannelRegistry().Register(event.WorldId, event.ChannelId, event.IpAddress, event.Port)
		} else if event.Status == EventStatusShutdown {
			l.Debugf("Unregistering channel [%d] for world [%d] at [%s:%d].", event.ChannelId, event.WorldId, event.IpAddress, event.Port)
			GetChannelRegistry().RemoveByWorldAndChannel(event.WorldId, event.ChannelId)
		} else {
			l.Errorf("Unhandled event status [%s].", event.Status)
		}
	}
}
