package character

import (
	consumer2 "atlas-maps/kafka/consumer"
	_map "atlas-maps/map"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
)

const consumerStatusEvent = "status_event"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerStatusEvent)(EnvEventTopicCharacterStatus)(groupId)
	}
}

func StatusEventLoginRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin))
}

func StatusEventLogoutRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout))
}

func StatusEventMapChangedRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged))
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLoginBody]) {
	if event.Type == EventCharacterStatusTypeLogin {
		l.Debugf("Character [%d] has logged in. worldId [%d] channelId [%d] mapId [%d].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.MapId)
		_map.Enter(l, ctx, event.Tenant)(event.WorldId, event.Body.ChannelId, event.Body.MapId, event.CharacterId)
		return
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLogoutBody]) {
	if event.Type == EventCharacterStatusTypeLogout {
		l.Debugf("Character [%d] has logged out. worldId [%d] channelId [%d] mapId [%d].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.MapId)
		_map.Exit(l, ctx, event.Tenant)(event.WorldId, event.Body.ChannelId, event.Body.MapId, event.CharacterId)
		return
	}
}

func handleStatusEventMapChanged(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventMapChangedBody]) {
	if event.Type == EventCharacterStatusTypeMapChanged {
		l.Debugf("Character [%d] has changed maps. worldId [%d] channelId [%d] oldMapId [%d] newMapId [%d].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.OldMapId, event.Body.TargetMapId)
		_map.Transition(l, ctx, event.Tenant)(event.WorldId, event.Body.ChannelId, event.Body.TargetMapId, event.CharacterId, event.Body.OldMapId)
	}
}
