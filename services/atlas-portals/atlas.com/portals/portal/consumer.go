package portal

import (
	consumer2 "atlas-portals/kafka/consumer"
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
			rf(consumer2.NewConfig(l)("portal_command")(EnvPortalCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvPortalCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEnterCommand)))
	}
}

func handleEnterCommand(l logrus.FieldLogger, ctx context.Context, command commandEvent[enterBody]) {
	l.Debugf("Received command for Character [%d] to enter portal [%d] in map [%d].", command.Body.CharacterId, command.PortalId, command.MapId)
	Enter(l)(ctx)(command.WorldId, command.ChannelId, command.MapId, command.PortalId, command.Body.CharacterId)
}
