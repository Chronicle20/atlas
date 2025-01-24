package npc

import (
	"atlas-npc-conversations/conversation"
	consumer2 "atlas-npc-conversations/kafka/consumer"
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
			rf(consumer2.NewConfig(l)("npc_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStartConversationCommand)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleContinueConversationCommand)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEndConversationCommand)))
	}
}

func handleStartConversationCommand(l logrus.FieldLogger, ctx context.Context, c command[startConversationCommandBody]) {
	if c.Type != CommandTypeStartConversation {
		return
	}
	_ = conversation.Start(l)(ctx)(c.Body.WorldId, c.Body.ChannelId, c.Body.MapId, c.NpcId, c.CharacterId)
}

func handleContinueConversationCommand(l logrus.FieldLogger, ctx context.Context, c command[continueConversationCommandBody]) {
	if c.Type != CommandTypeContinueConversation {
		return
	}
	_ = conversation.Continue(l)(ctx)(c.NpcId, c.CharacterId, c.Body.Action, c.Body.LastMessageType, c.Body.Selection)
}

func handleEndConversationCommand(l logrus.FieldLogger, ctx context.Context, c command[endConversationCommandBody]) {
	if c.Type != CommandTypeEndConversation {
		return
	}
	_ = conversation.End(l)(ctx)(c.CharacterId)
}
