package character

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

const consumerStatusEvent = "character_status"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerStatusEvent)(EnvEventTopicCharacterStatus)(groupId)
	}
}

func LoginStatusRegister(l logrus.FieldLogger) func(db *gorm.DB) (string, handler.Handler) {
	return func(db *gorm.DB) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin(db)))
	}
}

func handleStatusEventLogin(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLoginBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLoginBody]) {
		if event.Type != EventCharacterStatusTypeLogin {
			return
		}

		err := guild.UpdateMemberOnline(l)(ctx)(db)(event.CharacterId, true)
		if err != nil {
			l.WithError(err).Errorf("Unable to process login for character [%d].", event.CharacterId)
		}
	}
}

func LogoutStatusRegister(l logrus.FieldLogger) func(db *gorm.DB) (string, handler.Handler) {
	return func(db *gorm.DB) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout(db)))
	}
}

func handleStatusEventLogout(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLogoutBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLogoutBody]) {
		if event.Type != EventCharacterStatusTypeLogout {
			return

		}

		err := guild.UpdateMemberOnline(l)(ctx)(db)(event.CharacterId, false)
		if err != nil {
			l.WithError(err).Errorf("Unable to process logout for character [%d].", event.CharacterId)
		}
	}
}
