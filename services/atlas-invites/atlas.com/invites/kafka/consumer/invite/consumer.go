package invite

import (
	invite3 "atlas-invites/invite"
	consumer2 "atlas-invites/kafka/consumer"
	invite2 "atlas-invites/kafka/message/invite"
	"context"

	"github.com/Chronicle20/atlas-constants/invite"
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

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(invite2.EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRejectCommand))); err != nil {
			return err
		}
		return nil
	}
}

func handleCreateCommand(l logrus.FieldLogger, ctx context.Context, c invite2.Command[invite2.CreateCommandBody]) {
	if c.Type != invite.CommandTypeCreate {
		return
	}
	_, _ = invite3.NewProcessor(l, ctx).CreateAndEmit(uint32(c.Body.ReferenceId), c.WorldId, string(c.InviteType), uint32(c.Body.OriginatorId), uint32(c.Body.TargetId), c.TransactionId)
}

func handleAcceptCommand(l logrus.FieldLogger, ctx context.Context, c invite2.Command[invite2.AcceptCommandBody]) {
	if c.Type != invite.CommandTypeAccept {
		return
	}
	_, _ = invite3.NewProcessor(l, ctx).AcceptAndEmit(uint32(c.Body.ReferenceId), c.WorldId, string(c.InviteType), uint32(c.Body.TargetId), c.TransactionId)
}

func handleRejectCommand(l logrus.FieldLogger, ctx context.Context, c invite2.Command[invite2.RejectCommandBody]) {
	if c.Type != invite.CommandTypeReject {
		return
	}
	_, _ = invite3.NewProcessor(l, ctx).RejectAndEmit(uint32(c.Body.OriginatorId), c.WorldId, string(c.InviteType), uint32(c.Body.TargetId), c.TransactionId)
}
