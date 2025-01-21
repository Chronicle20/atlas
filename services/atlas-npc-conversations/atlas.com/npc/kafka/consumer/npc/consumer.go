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
	_ = conversation.Start(l)(ctx)(c.WorldId, c.ChannelId, c.MapId, c.NpcId, c.CharacterId)
}
