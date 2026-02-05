package portal

import (
	"atlas-portals/blocked"
	consumer2 "atlas-portals/kafka/consumer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
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
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleBlockCommand)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleUnblockCommand)))
	}
}

func handleEnterCommand(l logrus.FieldLogger, ctx context.Context, command commandEvent[enterBody]) {
	if command.Type != CommandTypeEnter {
		return
	}
	f := field.NewBuilder(command.WorldId, command.ChannelId, command.MapId).SetInstance(command.Instance).Build()
	l.Debugf("Received command for Character [%d] to enter portal [%d] in map [%d].", command.Body.CharacterId, command.PortalId, command.MapId)
	Enter(l)(ctx)(f, command.PortalId, command.Body.CharacterId)
}

func handleBlockCommand(l logrus.FieldLogger, ctx context.Context, command commandEvent[blockBody]) {
	if command.Type != CommandTypeBlock {
		return
	}
	t := tenant.MustFromContext(ctx)
	l.Debugf("Received command to block portal [%d] in map [%d] for character [%d].", command.PortalId, command.MapId, command.Body.CharacterId)
	blocked.GetCache().Block(t.Id(), command.Body.CharacterId, command.MapId, command.PortalId)
}

func handleUnblockCommand(l logrus.FieldLogger, ctx context.Context, command commandEvent[unblockBody]) {
	if command.Type != CommandTypeUnblock {
		return
	}
	t := tenant.MustFromContext(ctx)
	l.Debugf("Received command to unblock portal [%d] in map [%d] for character [%d].", command.PortalId, command.MapId, command.Body.CharacterId)
	blocked.GetCache().Unblock(t.Id(), command.Body.CharacterId, command.MapId, command.PortalId)
}
