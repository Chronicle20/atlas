package npc

import (
	"atlas-npc-conversations/conversation"
	consumer2 "atlas-npc-conversations/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
)

const consumerCommand = "npc_command"

func CommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerCommand)(EnvCommandTopic)(groupId)
	}
}

func StartConversationCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleStartConversationCommand))
}

func handleStartConversationCommand(l logrus.FieldLogger, ctx context.Context, c command[startConversationCommandBody]) {
	_ = conversation.Start(l)(ctx)(c.Body.WorldId, c.Body.ChannelId, c.Body.MapId, c.NpcId, c.CharacterId)
}

func ContinueConversationCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleContinueConversationCommand))
}

func handleContinueConversationCommand(l logrus.FieldLogger, ctx context.Context, c command[continueConversationCommandBody]) {
	_ = conversation.Continue(l)(ctx)(c.NpcId, c.CharacterId, c.Body.Action, c.Body.LastMessageType, c.Body.Selection)
}

func EndConversationCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleEndConversationCommand))
}

func handleEndConversationCommand(l logrus.FieldLogger, ctx context.Context, c command[endConversationCommandBody]) {
	_ = conversation.End(l)(ctx)(c.CharacterId)
}
