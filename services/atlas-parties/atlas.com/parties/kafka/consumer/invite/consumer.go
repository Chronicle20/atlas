package invite

import (
	consumer2 "atlas-parties/kafka/consumer"
	"atlas-parties/party"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
)

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)("invite_status_event")(EnvEventStatusTopic)(groupId)
	}
}

func AcceptedStatusEventRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventStatusTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleAcceptedStatusEvent))
}

func handleAcceptedStatusEvent(l logrus.FieldLogger, ctx context.Context, e statusEvent[acceptedEventBody]) {
	if e.Type != EventInviteStatusTypeAccepted {
		return
	}
	if e.InviteType != InviteTypeParty {
		return
	}

	_, err := party.Join(l)(ctx)(e.ReferenceId, e.Body.TargetId)
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to join party [%d].", e.Body.TargetId, e.ReferenceId)
	}

}
