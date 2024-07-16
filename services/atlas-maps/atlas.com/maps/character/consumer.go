package character

import (
	"atlas-maps/kafka"
	_map "atlas-maps/map"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const consumerStatusEvent = "status_event"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerStatusEvent)(EnvEventTopicCharacterStatus)(groupId)
	}
}

func StatusEventLoginRegister(l logrus.FieldLogger) (string, handler.Handler) {
	return kafka.LookupTopic(l)(EnvEventTopicCharacterStatus), message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin))
}

func StatusEventLogoutRegister(l logrus.FieldLogger) (string, handler.Handler) {
	return kafka.LookupTopic(l)(EnvEventTopicCharacterStatus), message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout))
}

func handleStatusEventLogin(l logrus.FieldLogger, span opentracing.Span, event statusEvent[statusEventLoginBody]) {
	if event.Type == EventCharacterStatusTypeLogin {
		l.Debugf("Received CharacterStatus [%s] Event. characterId [%d] worldId [%d] channelId [%d] mapId [%d].", event.Type, event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.MapId)
		_map.Enter(l, span, event.Tenant)(event.WorldId, event.Body.ChannelId, event.Body.MapId, event.CharacterId)
		return
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, span opentracing.Span, event statusEvent[statusEventLogoutBody]) {
	if event.Type == EventCharacterStatusTypeLogout {
		l.Debugf("Received CharacterStatus [%s] Event. characterId [%d] worldId [%d] channelId [%d] mapId [%d].", event.Type, event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.MapId)
		_map.Exit(l, span, event.Tenant)(event.WorldId, event.Body.ChannelId, event.Body.MapId, event.CharacterId)
		return
	}
}
