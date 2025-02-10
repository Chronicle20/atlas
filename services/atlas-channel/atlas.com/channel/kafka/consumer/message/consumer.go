package message

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	message2 "atlas-channel/message"
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

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("chat_event")(EnvEventTopicChat)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvEventTopicChat)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleGeneralChat(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMultiChat(sc, wp))))
			}
		}
	}
}

func handleGeneralChat(sc server.Model, wp writer.Producer) message.Handler[chatEvent[generalChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event chatEvent[generalChatBody]) {
		if event.Type != ChatTypeGeneral {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), event.WorldId, event.ChannelId) {
			return
		}

		c, err := character.GetById(l)(ctx)()(event.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] chatting.", event.CharacterId)
			return
		}

		_ = _map.ForSessionsInMap(l)(ctx)(event.WorldId, event.ChannelId, event.MapId, showGeneralChatForSession(l)(ctx)(wp)(event, c.Gm()))
	}
}

func showGeneralChatForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event chatEvent[generalChatBody], gm bool) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event chatEvent[generalChatBody], gm bool) model.Operator[session.Model] {
		return func(wp writer.Producer) func(event chatEvent[generalChatBody], gm bool) model.Operator[session.Model] {
			generalChatFunc := session.Announce(l)(ctx)(wp)(writer.CharacterGeneralChat)
			return func(event chatEvent[generalChatBody], gm bool) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := generalChatFunc(s, writer.CharacterGeneralChatBody(event.CharacterId, gm, event.Message, event.Body.BalloonOnly))
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

func handleMultiChat(sc server.Model, wp writer.Producer) message.Handler[chatEvent[multiChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e chatEvent[multiChatBody]) {
		if e.Type == ChatTypeGeneral {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		c, err := character.GetById(l)(ctx)()(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] chatting.", e.CharacterId)
			return
		}

		for _, cid := range e.Body.Recipients {
			session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(cid, sendMultiChat(l)(ctx)(wp)(c.Name(), e.Message, message2.MultiChatTypeStrToInd(e.Type)))
		}
	}
}

func sendMultiChat(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(name string, message string, mode byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(name string, message string, mode byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(name string, message string, mode byte) model.Operator[session.Model] {
			multiChatFunc := session.Announce(l)(ctx)(wp)(writer.CharacterMultiChat)
			return func(name string, message string, mode byte) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := multiChatFunc(s, writer.CharacterMultiChatBody(name, message, mode))
					if err != nil {
						return err
					}
					return nil
				}
			}
		}
	}
}
