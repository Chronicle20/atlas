package invite

import (
	consumer2 "atlas-messengers/kafka/consumer"
	messageInvite "atlas-messengers/kafka/message/invite"
	"atlas-messengers/messenger"
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
			rf(consumer2.NewConfig(l)("invite_status_event")(messageInvite.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(messageInvite.EnvEventStatusTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptedStatusEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleAcceptedStatusEvent(l logrus.FieldLogger, ctx context.Context, e messageInvite.StatusEvent[messageInvite.AcceptedEventBody]) {
	if e.Type != invite.StatusTypeAccepted {
		return
	}
	if e.InviteType != invite.TypeMessenger {
		return
	}

	_, err := messenger.Join(l)(ctx)(e.TransactionId, uint32(e.ReferenceId), uint32(e.Body.TargetId))
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to join messenger [%d].", e.Body.TargetId, e.ReferenceId)
	}
}
