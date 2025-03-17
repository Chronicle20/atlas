package character

import (
	consumer2 "atlas-consumables/kafka/consumer"
	"atlas-consumables/map/character"
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
			rf(consumer2.NewConfig(l)("status_event")(EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged)))
	}
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLoginBody]) {
	if event.Type == EventCharacterStatusTypeLogin {
		l.Debugf("Character [%d] has logged in. worldId [%d] channelId [%d] mapId [%d].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.MapId)
		character.Enter(ctx)(event.WorldId, event.Body.ChannelId, event.Body.MapId, event.CharacterId)
		return
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLogoutBody]) {
	if event.Type == EventCharacterStatusTypeLogout {
		l.Debugf("Character [%d] has logged out. worldId [%d] channelId [%d] mapId [%d].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.MapId)
		character.Exit(ctx)(event.WorldId, event.Body.ChannelId, event.Body.MapId, event.CharacterId)
		return
	}
}

func handleStatusEventMapChanged(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventMapChangedBody]) {
	if event.Type == EventCharacterStatusTypeMapChanged {
		l.Debugf("Character [%d] has changed maps. worldId [%d] channelId [%d] oldMapId [%d] newMapId [%d].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.OldMapId, event.Body.TargetMapId)
		character.TransitionMap(ctx)(event.WorldId, event.Body.ChannelId, event.Body.TargetMapId, event.CharacterId, event.Body.OldMapId)
	}
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, event statusEvent[changeChannelEventLoginBody]) {
	if event.Type == EventCharacterStatusTypeChannelChanged {
		l.Debugf("Character [%d] has changed channels. worldId [%d] channelId [%d] oldChannelId [%d].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.OldChannelId)
		character.TransitionChannel(ctx)(event.WorldId, event.Body.ChannelId, event.Body.OldChannelId, event.CharacterId, event.Body.MapId)
	}
}
