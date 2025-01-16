package invite

import (
	consumer2 "atlas-buddies/kafka/consumer"
	"atlas-buddies/list"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
	if e.InviteType != InviteTypeBuddy {
		return
	}
}

func RejectedStatusEventRegister(l logrus.FieldLogger) func(db *gorm.DB) (string, handler.Handler) {
	return func(db *gorm.DB) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventStatusTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleRejectedStatusEvent(db)))
	}
}

func handleRejectedStatusEvent(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, e statusEvent[rejectedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[rejectedEventBody]) {
		if e.Type != EventInviteStatusTypeRejected {
			return
		}
		if e.InviteType != InviteTypeBuddy {
			return
		}

		_ = list.Delete(l)(ctx)(db)(e.Body.OriginatorId, e.WorldId, e.Body.TargetId)
	}
}
