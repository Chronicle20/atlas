package invite

import (
	consumer2 "atlas-invites/kafka/consumer"
	"atlas-invites/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
)

func CommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)("invite_command")(EnvCommandTopic)(groupId)
	}
}

func CreateCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleCreateCommand))
}

func handleCreateCommand(l logrus.FieldLogger, ctx context.Context, c commandEvent[createCommandBody]) {
	_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(createdStatusEventProvider(c.Body.ReferenceId, c.WorldId, c.Body.InviteType, c.Body.OriginatorId, c.Body.TargetId))
}
