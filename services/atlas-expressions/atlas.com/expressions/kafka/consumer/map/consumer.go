package _map

import (
	"atlas-expressions/expression"
	consumer2 "atlas-expressions/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
)

const consumerStatusEvent = "status_event"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerStatusEvent)(EnvEventTopicMapStatus)(groupId)
	}
}

func StatusEventCharacterExitRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicMapStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit))
}

func handleStatusEventCharacterExit(l logrus.FieldLogger, ctx context.Context, event statusEvent[characterExit]) {
	if event.Type != EventTopicMapStatusTypeCharacterExit {
		return
	}
	_ = expression.Clear(l)(ctx)(event.Body.CharacterId)
}
