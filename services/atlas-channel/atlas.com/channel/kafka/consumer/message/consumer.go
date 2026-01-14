package message

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	message3 "atlas-channel/kafka/message/message"
	_map "atlas-channel/map"
	message2 "atlas-channel/message"
	"atlas-channel/pet"
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
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("chat_event")(message3.EnvEventTopicChat)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(message3.EnvEventTopicChat)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleGeneralChat(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMultiChat(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleWhisperChat(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMessengerChat(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handlePetChat(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handlePinkChat(sc, wp))))
			}
		}
	}
}

func handleGeneralChat(sc server.Model, wp writer.Producer) message.Handler[message3.ChatEvent[message3.GeneralChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e message3.ChatEvent[message3.GeneralChatBody]) {
		if e.Type != message3.ChatTypeGeneral {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		c, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] chatting.", e.ActorId)
			return
		}

		err = _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Map(_map2.Id(e.MapId)), showGeneralChatForSession(l)(ctx)(wp)(e, c.Gm()))
		if err != nil {
			l.WithError(err).Errorf("Unable to send message from character [%d] to map [%d].", e.ActorId, e.MapId)
		}
	}
}

func showGeneralChatForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event message3.ChatEvent[message3.GeneralChatBody], gm bool) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event message3.ChatEvent[message3.GeneralChatBody], gm bool) model.Operator[session.Model] {
		return func(wp writer.Producer) func(event message3.ChatEvent[message3.GeneralChatBody], gm bool) model.Operator[session.Model] {
			return func(event message3.ChatEvent[message3.GeneralChatBody], gm bool) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterChatGeneral)(writer.CharacterChatGeneralBody(event.ActorId, gm, event.Message, event.Body.BalloonOnly))
			}
		}
	}
}

func handleMultiChat(sc server.Model, wp writer.Producer) message.Handler[message3.ChatEvent[message3.MultiChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e message3.ChatEvent[message3.MultiChatBody]) {
		if e.Type == message3.ChatTypeGeneral || e.Type == message3.ChatTypeWhisper {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		c, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] chatting.", e.ActorId)
			return
		}

		for _, cid := range e.Body.Recipients {
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(cid, sendMultiChat(l)(ctx)(wp)(c.Name(), e.Message, message2.MultiChatTypeStrToInd(e.Type)))
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

func handleWhisperChat(sc server.Model, wp writer.Producer) message.Handler[message3.ChatEvent[message3.WhisperChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e message3.ChatEvent[message3.WhisperChatBody]) {
		if e.Type != message3.ChatTypeWhisper {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		c, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] sending whisper.", e.ActorId)
			return
		}
		tc, err := character.NewProcessor(l, ctx).GetById()(e.Body.Recipient)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] receiving whisper.", e.Body.Recipient)
			return
		}

		bp := writer.CharacterChatWhisperSendResultBody(tc, true)
		err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.ActorId, session.Announce(l)(ctx)(wp)(writer.CharacterChatWhisper)(bp))
		if err != nil {
			l.WithError(err).Errorf("Unable to send whisper message from [%d] to [%d].", e.ActorId, e.Body.Recipient)
		}

		bp = writer.CharacterChatWhisperReceiptBody(c, e.ChannelId, e.Message)
		err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.Body.Recipient, session.Announce(l)(ctx)(wp)(writer.CharacterChatWhisper)(bp))
		if err != nil {
			l.WithError(err).Errorf("Unable to send whisper message from [%d] to [%d].", e.ActorId, e.Body.Recipient)
		}
	}
}

func handleMessengerChat(sc server.Model, wp writer.Producer) message.Handler[message3.ChatEvent[message3.MessengerChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e message3.ChatEvent[message3.MessengerChatBody]) {
		if e.Type != message3.ChatTypeMessenger {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		for _, cid := range e.Body.Recipients {
			bp := session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationChatBody(l)(e.Message))
			err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(cid, bp)
			if err != nil {
				l.WithError(err).Errorf("Unable to send message of type [%s] to character [%d].", e.Type, cid)
			}
		}
	}
}

func handlePetChat(sc server.Model, wp writer.Producer) message.Handler[message3.ChatEvent[message3.PetChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e message3.ChatEvent[message3.PetChatBody]) {
		if e.Type != message3.ChatTypePet {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.WorldId(), sc.ChannelId())(e.Body.OwnerId)
		if err != nil {
			return
		}

		p := pet.NewModelBuilder(e.ActorId, 0, 0, "").SetOwnerID(e.Body.OwnerId).SetSlot(e.Body.PetSlot).MustBuild()
		_ = _map.NewProcessor(l, ctx).ForSessionsInMap(s.Map(), session.Announce(l)(ctx)(wp)(writer.PetChat)(writer.PetChatBody(p, e.Body.Type, e.Body.Action, e.Message, e.Body.Balloon)))
	}
}

func handlePinkChat(sc server.Model, wp writer.Producer) message.Handler[message3.ChatEvent[message3.PinkTextChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e message3.ChatEvent[message3.PinkTextChatBody]) {
		if e.Type != message3.ChatTypePinkText {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		characterName := ""

		c, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err == nil {
			characterName = c.Name()
		}

		// TODO retrieve medal name
		for _, cid := range e.Body.Recipients {
			bp := session.Announce(l)(ctx)(wp)(writer.WorldMessage)(writer.WorldMessagePinkTextBody(l, sc.Tenant())("", characterName, e.Message))
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(cid, bp)
			if err != nil {
				l.WithError(err).Errorf("Unable to send message of type [%s] to character [%d].", e.Type, cid)
			}
		}
	}
}
