package channel

import (
	"atlas-channel/channel"
	consumer2 "atlas-channel/kafka/consumer"
	channel2 "atlas-channel/kafka/message/channel"
	"atlas-channel/listener"
	"atlas-channel/server"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("channel_service_command")(channel2.EnvCommandTopicChannelStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(ipAddress string, port int) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(ipAddress string, port int) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(ipAddress string, port int) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(channel2.EnvCommandTopicChannelStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandStatus(sc, ipAddress, port))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleCommandStatus(sc server.Model, ipAddress string, port int) message.Handler[channel2.ChannelStatusCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c channel2.ChannelStatusCommand) {
		st := sc.Tenant()
		mt := tenant.MustFromContext(ctx)
		if mt.Id() == st.Id() {
			err := channel.NewProcessor(l, ctx).Register(sc.Channel(), ipAddress, port)
			if err != nil {
				l.WithError(err).Errorf("Unable to respond to world service status command. World service will not know about this channel.")
			}
		}
	}
}
