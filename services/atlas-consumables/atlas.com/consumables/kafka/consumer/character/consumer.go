package character

import (
	consumer2 "atlas-consumables/kafka/consumer"
	character2 "atlas-consumables/kafka/message/character"
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
			rf(consumer2.NewConfig(l)("status_event")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged)))
	}
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.StatusEventLoginBody]) {
	if event.Type == character2.EventCharacterStatusTypeLogin {
		l.Debugf("Character [%d] has logged in. worldId [%d] channelId [%d] mapId [%d].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.MapId)
		character.NewProcessor(l, ctx).Enter(event.WorldId, event.Body.ChannelId, event.Body.MapId, event.CharacterId)
		return
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.StatusEventLogoutBody]) {
	if event.Type == character2.EventCharacterStatusTypeLogout {
		l.Debugf("Character [%d] has logged out. worldId [%d] channelId [%d] mapId [%d].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.MapId)
		character.NewProcessor(l, ctx).Exit(event.WorldId, event.Body.ChannelId, event.Body.MapId, event.CharacterId)
		return
	}
}

func handleStatusEventMapChanged(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.StatusEventMapChangedBody]) {
	if event.Type == character2.EventCharacterStatusTypeMapChanged {
		l.Debugf("Character [%d] has changed maps. worldId [%d] channelId [%d] oldMapId [%d] newMapId [%d].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.OldMapId, event.Body.TargetMapId)
		character.NewProcessor(l, ctx).TransitionMap(event.WorldId, event.Body.ChannelId, event.Body.TargetMapId, event.CharacterId, event.Body.OldMapId)
	}
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.ChangeChannelEventLoginBody]) {
	if event.Type == character2.EventCharacterStatusTypeChannelChanged {
		l.Debugf("Character [%d] has changed channels. worldId [%d] channelId [%d] oldChannelId [%d].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.OldChannelId)
		character.NewProcessor(l, ctx).TransitionChannel(event.WorldId, event.Body.ChannelId, event.Body.OldChannelId, event.CharacterId, event.Body.MapId)
	}
}
