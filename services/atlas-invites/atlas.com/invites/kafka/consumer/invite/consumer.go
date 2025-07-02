package invite

import (
	invite3 "atlas-invites/invite"
	consumer2 "atlas-invites/kafka/consumer"
	invite2 "atlas-invites/kafka/message/invite"
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
			rf(consumer2.NewConfig(l)("invite_command")(invite2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(invite2.EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateCommand)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptCommand)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRejectCommand)))
	}
}

func handleCreateCommand(l logrus.FieldLogger, ctx context.Context, c invite2.CommandEvent[invite2.CreateCommandBody]) {
	if c.Type != invite2.CommandInviteTypeCreate {
		return
	}
	_ = invite3.NewProcessor(l, ctx).Create(c.Body.ReferenceId, c.WorldId, c.InviteType, c.Body.OriginatorId, c.Body.TargetId)
}

func handleAcceptCommand(l logrus.FieldLogger, ctx context.Context, c invite2.CommandEvent[invite2.AcceptCommandBody]) {
	if c.Type != invite2.CommandInviteTypeAccept {
		return
	}
	_ = invite3.NewProcessor(l, ctx).Accept(c.Body.ReferenceId, c.WorldId, c.InviteType, c.Body.TargetId)
}

func handleRejectCommand(l logrus.FieldLogger, ctx context.Context, c invite2.CommandEvent[invite2.RejectCommandBody]) {
	if c.Type != invite2.CommandInviteTypeReject {
		return
	}
	_ = invite3.NewProcessor(l, ctx).Reject(c.Body.OriginatorId, c.WorldId, c.InviteType, c.Body.TargetId)
}
