package message

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const (
	generalChatEventConsumer = "general_chat_consumer"
)

func GeneralChatEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(generalChatEventConsumer)(EnvEventTopicGeneralChat)(groupId)
	}
}

func GeneralChatEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicGeneralChat)()
		return t, message.AdaptHandler(message.PersistentConfig(handleGeneralChat(sc, wp)))
	}
}

func handleGeneralChat(sc server.Model, wp writer.Producer) message.Handler[generalChatEvent] {
	return func(l logrus.FieldLogger, span opentracing.Span, event generalChatEvent) {
		if !sc.Is(event.Tenant, event.WorldId, event.ChannelId) {
			return
		}

		c, err := character.GetById(l, span, event.Tenant)(event.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] chatting.", event.CharacterId)
			return
		}

		_ = _map.ForOtherSessionsInMap(l, span, event.Tenant)(event.WorldId, event.ChannelId, event.MapId, event.CharacterId, showGeneralChatForSession(l, wp)(event, c.Gm()))
	}
}

func showGeneralChatForSession(l logrus.FieldLogger, wp writer.Producer) func(event generalChatEvent, gm bool) model.Operator[session.Model] {
	generalChatFunc := session.Announce(l)(wp)(writer.CharacterGeneralChat)
	return func(event generalChatEvent, gm bool) model.Operator[session.Model] {
		return func(s session.Model) error {
			err := generalChatFunc(s, writer.CharacterGeneralChatBody(event.CharacterId, gm, event.Message, event.BalloonOnly))
			if err != nil {
				l.WithError(err).Errorf("Unable to write message to rest of map.")
				return err
			}
			return nil
		}
	}
}
