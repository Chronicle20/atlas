package character

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
		err := list.UpdateChannel(l)(ctx)(db)(event.CharacterId, event.WorldId, int8(event.Body.ChannelId))
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
		err := list.UpdateChannel(l)(ctx)(db)(event.CharacterId, event.WorldId, -1)
		if err != nil {
			l.WithError(err).Errorf("Unable to process logout for character [%d].", event.CharacterId)
		}
	}
}

func ChannelChangedStatusRegister(l logrus.FieldLogger) func(db *gorm.DB) (string, handler.Handler) {
	return func(db *gorm.DB) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged(db)))
	}
}

func handleStatusEventChannelChanged(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventChannelChangedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventChannelChangedBody]) {
		if event.Type != EventCharacterStatusTypeChannelChanged {
			return
		}
		if event.Body.ChannelId == event.Body.OldChannelId {
			return
		}
		err := list.UpdateChannel(l)(ctx)(db)(event.CharacterId, event.WorldId, int8(event.Body.ChannelId))
		if err != nil {
			l.WithError(err).Errorf("Unable to process change channel for character [%d].", event.CharacterId)
		}
	}
}
