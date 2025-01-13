package character

import (
	consumer2 "atlas-parties/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
)

const consumerStatusEvent = "character_status"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerStatusEvent)(EnvEventTopicCharacterStatus)(groupId)
	}
}

func LoginStatusRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin))
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLoginBody]) {
	if event.Type != EventCharacterStatusTypeLogin {
		return
	}
	err := Login(l)(ctx)(event.WorldId, event.Body.ChannelId, event.Body.MapId, event.CharacterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to process login for character [%d].", event.CharacterId)
	}
}

func LogoutStatusRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout))
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLogoutBody]) {
	if event.Type != EventCharacterStatusTypeLogout {
		return
	}
	err := Logout(l)(ctx)(event.CharacterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to process logout for character [%d].", event.CharacterId)
	}
}

func MapChangedStatusRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleMapChangedStatusEventLogout))
}

func handleMapChangedStatusEventLogout(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventMapChangedBody]) {
	if event.Type != EventCharacterStatusTypeMapChanged {
		return
	}
	err := MapChange(l)(ctx)(event.CharacterId, event.Body.TargetMapId)
	if err != nil {
		l.WithError(err).Errorf("Unable to process map changed for character [%d].", event.CharacterId)
	}
}
