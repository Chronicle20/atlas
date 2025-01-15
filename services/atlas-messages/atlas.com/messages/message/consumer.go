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
	chatEventConsumer = "chat_consumer"
)

func ChatCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(chatEventConsumer)(EnvCommandTopicChat)(groupId)
	}
}

func GeneralChatCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopicChat)()
	return t, message.AdaptHandler(message.PersistentConfig(handleGeneralChat))
}

func handleGeneralChat(l logrus.FieldLogger, ctx context.Context, event chatCommand[generalChatBody]) {
	if event.Type != ChatTypeGeneral {
		return
	}
	_ = HandleGeneral(l)(ctx)(event.WorldId, event.ChannelId, event.MapId, event.CharacterId, event.Message, event.Body.BalloonOnly)
}

func MultiChatCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopicChat)()
	return t, message.AdaptHandler(message.PersistentConfig(handleMultiChat))
}

func handleMultiChat(l logrus.FieldLogger, ctx context.Context, e chatCommand[multiChatBody]) {
	if e.Type == ChatTypeGeneral {
		return
	}
	_ = HandleMulti(l)(ctx)(e.WorldId, e.ChannelId, e.MapId, e.CharacterId, e.Message, e.Type, e.Body.Recipients)
}
