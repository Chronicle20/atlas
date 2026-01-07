package system_message

import (
	consumer2 "atlas-channel/kafka/consumer"
	system_message2 "atlas-channel/kafka/message/system_message"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("system_message_command")(system_message2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(system_message2.EnvCommandTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSendMessage(sc, wp))))
			}
		}
	}
}

func handleSendMessage(sc server.Model, wp writer.Producer) message.Handler[system_message2.Command[system_message2.SendMessageBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd system_message2.Command[system_message2.SendMessageBody]) {
		if cmd.Type != system_message2.CommandSendMessage {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, world.Id(cmd.WorldId), channel.Id(cmd.ChannelId)) {
			return
		}

		// Map message type to body producer
		var bodyProducer writer.BodyProducer
		switch cmd.Body.MessageType {
		case "NOTICE":
			bodyProducer = writer.WorldMessageNoticeBody(l, sc.Tenant())(cmd.Body.Message)
		case "POP_UP":
			bodyProducer = writer.WorldMessagePopUpBody(l, sc.Tenant())(cmd.Body.Message)
		case "PINK_TEXT":
			bodyProducer = writer.WorldMessagePinkTextBody(l, sc.Tenant())("", "", cmd.Body.Message)
		case "BLUE_TEXT":
			bodyProducer = writer.WorldMessageBlueTextBody(l, sc.Tenant())("", "", cmd.Body.Message)
		default:
			l.Warnf("Unknown message type: %s, defaulting to PINK_TEXT", cmd.Body.MessageType)
			bodyProducer = writer.WorldMessagePinkTextBody(l, sc.Tenant())("", "", cmd.Body.Message)
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.WorldMessage)(bodyProducer))
		if err != nil {
			l.WithError(err).Errorf("Unable to send message to character [%d].", cmd.CharacterId)
		}
	}
}
