package character

import (
	"atlas-parties/character"
	consumer2 "atlas-parties/kafka/consumer"
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
			rf(consumer2.NewConfig(l)("character_status_event")(EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMapChangedStatusEventLogout)))
	}
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLoginBody]) {
	if event.Type != EventCharacterStatusTypeLogin {
		return
	}
	err := character.Login(l)(ctx)(event.WorldId, event.Body.ChannelId, event.Body.MapId, event.CharacterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to process login for character [%d].", event.CharacterId)
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLogoutBody]) {
	if event.Type != EventCharacterStatusTypeLogout {
		return
	}
	err := character.Logout(l)(ctx)(event.CharacterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to process logout for character [%d].", event.CharacterId)
	}
}

func handleMapChangedStatusEventLogout(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventMapChangedBody]) {
	if event.Type != EventCharacterStatusTypeMapChanged {
		return
	}
	err := character.MapChange(l)(ctx)(event.CharacterId, event.Body.TargetMapId)
	if err != nil {
		l.WithError(err).Errorf("Unable to process map changed for character [%d].", event.CharacterId)
	}
}
