package _map

import (
	consumer2 "atlas-monsters/kafka/consumer"
	"atlas-monsters/monster"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const consumerStatusEvent = "status_event"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerStatusEvent)(EnvEventTopicMapStatus)(groupId)
	}
}

func StatusEventCharacterEnterRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicMapStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterEnter))
}

func StatusEventCharacterExitRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicMapStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit))
}

func handleStatusEventCharacterEnter(l logrus.FieldLogger, span opentracing.Span, event statusEvent[characterEnter]) {
	if event.Type != EventTopicMapStatusTypeCharacterEnter {
		return
	}

	ms, err := monster.GetInMap(l, span, event.Tenant)(event.WorldId, event.ChannelId, event.MapId)
	if err != nil {
		l.WithError(err).Errorf("Unable to retrieve monsters in map.")
		return
	}

	for _, m := range ms {
		if m.ControlCharacterId() == 0 {
			monster.FindNextController(l, span, event.Tenant)(m.UniqueId())
		}
	}
}

func handleStatusEventCharacterExit(l logrus.FieldLogger, span opentracing.Span, event statusEvent[characterExit]) {
	if event.Type != EventTopicMapStatusTypeCharacterExit {
		return
	}

	ms, err := monster.GetInMap(l, span, event.Tenant)(event.WorldId, event.ChannelId, event.MapId)
	if err != nil {
		l.WithError(err).Errorf("Unable to retrieve monsters in map.")
		return
	}
	for _, m := range ms {
		if m.ControlCharacterId() == event.Body.CharacterId {
			_, _ = monster.StopControl(l, span, event.Tenant)(m.UniqueId())
		}
	}
	for _, m := range ms {
		if m.ControlCharacterId() == event.Body.CharacterId {
			monster.FindNextController(l, span, event.Tenant)(m.UniqueId())
		}
	}
}
