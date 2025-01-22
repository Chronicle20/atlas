package invite

import (
	"atlas-guilds/guild"
	consumer2 "atlas-guilds/kafka/consumer"
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

func AcceptedStatusEventRegister(l logrus.FieldLogger) func(db *gorm.DB) (string, handler.Handler) {
	return func(db *gorm.DB) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventStatusTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[acceptedEventBody]) {
			if e.Type != EventInviteStatusTypeAccepted {
				return
			}
			if e.InviteType != InviteTypeGuild {
				return
			}

			err := guild.Join(l)(ctx)(db)(e.ReferenceId, e.Body.TargetId)
			if err != nil {
				l.WithError(err).Errorf("Character [%d] unable to join party [%d].", e.Body.TargetId, e.ReferenceId)
			}
		}))
	}
}
