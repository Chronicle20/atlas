package channel

import (
	"atlas-channel/kafka"
	"atlas-channel/server"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const (
	consumerNameStatus = "channel_service_command"
)

func CommandStatusConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerNameStatus)(EnvCommandTopicChannelStatus)(groupId)
	}
}

func CommandStatusRegister(sc server.Model, ipAddress string, port string) func(l *logrus.Logger) (string, handler.Handler) {
	return func(l *logrus.Logger) (string, handler.Handler) {
		return kafka.LookupTopic(l)(EnvCommandTopicChannelStatus), message.AdaptHandler(message.PersistentConfig(handleCommandStatus(sc, ipAddress, port)))
	}
}

func handleCommandStatus(sc server.Model, ipAddress string, port string) message.Handler[channelStatusCommand] {
	return func(l logrus.FieldLogger, span opentracing.Span, c channelStatusCommand) {
		if c.Tenant.Id == sc.Tenant().Id {
			err := Register(l, span, c.Tenant)(sc.WorldId(), sc.ChannelId(), ipAddress, port)
			if err != nil {
				l.WithError(err).Errorf("Unable to respond to world service status command. World service will not know about this channel.")
			}
		}
	}
}
