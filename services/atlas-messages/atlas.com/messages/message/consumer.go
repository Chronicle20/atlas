package message

import (
	consumer2 "atlas-messages/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
)

const (
	generalChatEventConsumer = "general_chat_consumer"
)

func GeneralChatCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(generalChatEventConsumer)(EnvCommandTopicGeneralChat)(groupId)
	}
}

func GeneralChatCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopicGeneralChat)()
	return t, message.AdaptHandler(message.PersistentConfig(handleGeneralChat))
}

func handleGeneralChat(l logrus.FieldLogger, ctx context.Context, event generalChatCommand) {
	_ = Handle(l, ctx, event.Tenant)(event.WorldId, event.ChannelId, event.MapId, event.CharacterId, event.Message, event.BalloonOnly)
}
