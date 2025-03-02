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
	"github.com/Chronicle20/atlas-constants/channel"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleWhisperChat(sc, wp))))
			}
		}
	}
}

func handleGeneralChat(sc server.Model, wp writer.Producer) message.Handler[chatEvent[generalChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e chatEvent[generalChatBody]) {
		if e.Type != ChatTypeGeneral {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		c, err := character.GetById(l)(ctx)()(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] chatting.", e.CharacterId)
			return
		}

		err = _map.ForSessionsInMap(l)(ctx)(sc.Map(_map2.Id(e.MapId)), showGeneralChatForSession(l)(ctx)(wp)(e, c.Gm()))
		if err != nil {
			l.WithError(err).Errorf("Unable to send message from character [%d] to map [%d].", e.CharacterId, e.MapId)
		}
	}
}

func showGeneralChatForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event chatEvent[generalChatBody], gm bool) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event chatEvent[generalChatBody], gm bool) model.Operator[session.Model] {
		return func(wp writer.Producer) func(event chatEvent[generalChatBody], gm bool) model.Operator[session.Model] {
			return func(event chatEvent[generalChatBody], gm bool) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterChatGeneral)(writer.CharacterChatGeneralBody(event.CharacterId, gm, event.Message, event.Body.BalloonOnly))
			}
		}
	}
}

func handleMultiChat(sc server.Model, wp writer.Producer) message.Handler[chatEvent[multiChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e chatEvent[multiChatBody]) {
		if e.Type == ChatTypeGeneral || e.Type == ChatTypeWhisper {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		c, err := character.GetById(l)(ctx)()(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] chatting.", e.CharacterId)
			return
		}

		for _, cid := range e.Body.Recipients {
			err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(cid, sendMultiChat(l)(ctx)(wp)(c.Name(), e.Message, message2.MultiChatTypeStrToInd(e.Type)))
			if err != nil {
				l.WithError(err).Errorf("Unable to send message of type [%s] to character [%d].", e.Type, cid)
			}
		}
	}
}

func sendMultiChat(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(name string, message string, mode byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(name string, message string, mode byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(name string, message string, mode byte) model.Operator[session.Model] {
			return func(name string, message string, mode byte) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterChatMulti)(writer.CharacterChatMultiBody(name, message, mode))
			}
		}
	}
}

func handleWhisperChat(sc server.Model, wp writer.Producer) message.Handler[chatEvent[whisperChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e chatEvent[whisperChatBody]) {
		if e.Type != ChatTypeWhisper {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		c, err := character.GetById(l)(ctx)()(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] sending whisper.", e.CharacterId)
			return
		}
		tc, err := character.GetById(l)(ctx)()(e.Body.Recipient)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] receiving whisper.", e.Body.Recipient)
			return
		}

		bp := writer.CharacterChatWhisperSendResultBody(tc, true)
		err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, session.Announce(l)(ctx)(wp)(writer.CharacterChatWhisper)(bp))
		if err != nil {
			l.WithError(err).Errorf("Unable to send whisper message from [%d] to [%d].", e.CharacterId, e.Body.Recipient)
		}

		bp = writer.CharacterChatWhisperReceiptBody(c, e.ChannelId, e.Message)
		err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.Recipient, session.Announce(l)(ctx)(wp)(writer.CharacterChatWhisper)(bp))
		if err != nil {
			l.WithError(err).Errorf("Unable to send whisper message from [%d] to [%d].", e.CharacterId, e.Body.Recipient)
		}
	}
}
