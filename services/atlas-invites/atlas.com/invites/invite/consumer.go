package invite

import (
	consumer2 "atlas-invites/kafka/consumer"
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
	if c.Type != CommandInviteTypeCreate {
		return
	}
	_ = Create(l)(ctx)(c.Body.ReferenceId, c.WorldId, c.InviteType, c.Body.OriginatorId, c.Body.TargetId)
}

func AcceptCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleAcceptCommand))
}

func handleAcceptCommand(l logrus.FieldLogger, ctx context.Context, c commandEvent[acceptCommandBody]) {
	if c.Type != CommandInviteTypeAccept {
		return
	}
	_ = Accept(l)(ctx)(c.Body.ReferenceId, c.WorldId, c.InviteType, c.Body.TargetId)
}

func RejectCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleRejectCommand))
}

func handleRejectCommand(l logrus.FieldLogger, ctx context.Context, c commandEvent[rejectCommandBody]) {
	if c.Type != CommandInviteTypeReject {
		return
	}
	_ = Reject(l)(ctx)(c.Body.OriginatorId, c.WorldId, c.InviteType, c.Body.TargetId)
}
