package invite

import (
	consumer2 "atlas-parties/kafka/consumer"
	"atlas-parties/party"
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
			rf(consumer2.NewConfig(l)("invite_status_event")(EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvEventStatusTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptedStatusEvent)))
	}
}

func handleAcceptedStatusEvent(l logrus.FieldLogger, ctx context.Context, e statusEvent[acceptedEventBody]) {
	if e.Type != EventInviteStatusTypeAccepted {
		return
	}
	if e.InviteType != InviteTypeParty {
		return
	}

	_, err := party.NewProcessor(l, ctx).Join(e.ReferenceId, e.Body.TargetId)
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to join party [%d].", e.Body.TargetId, e.ReferenceId)
	}

}
