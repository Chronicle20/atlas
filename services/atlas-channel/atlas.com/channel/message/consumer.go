package message

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
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
	return func(l logrus.FieldLogger, ctx context.Context, event generalChatEvent) {
		if !sc.Is(tenant.MustFromContext(ctx), event.WorldId, event.ChannelId) {
			return
		}

		c, err := character.GetById(l)(ctx)(event.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] chatting.", event.CharacterId)
			return
		}

		_ = _map.ForSessionsInMap(l)(ctx)(event.WorldId, event.ChannelId, event.MapId, showGeneralChatForSession(l)(ctx)(wp)(event, c.Gm()))
	}
}

func showGeneralChatForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event generalChatEvent, gm bool) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event generalChatEvent, gm bool) model.Operator[session.Model] {
		return func(wp writer.Producer) func(event generalChatEvent, gm bool) model.Operator[session.Model] {
			generalChatFunc := session.Announce(l)(ctx)(wp)(writer.CharacterGeneralChat)
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
	}
}
