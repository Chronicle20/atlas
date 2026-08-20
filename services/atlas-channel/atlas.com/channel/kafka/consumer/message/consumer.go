package message

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	message3 "atlas-channel/kafka/message/message"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	message2 "atlas-channel/message"
	"atlas-channel/pet"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	messengerpkt "github.com/Chronicle20/atlas/libs/atlas-packet/messenger"
	messengercb "github.com/Chronicle20/atlas/libs/atlas-packet/messenger/clientbound"
	petpkt "github.com/Chronicle20/atlas/libs/atlas-packet/pet/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("chat_event")(message3.EnvEventTopicChat)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(message3.EnvEventTopicChat)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleGeneralChat(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleMultiChat(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleWhisperChat(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleMessengerChat(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handlePetChat(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handlePinkChat(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleGeneralChat(sc server.Model, wp writer.Producer) message.Handler[message3.ChatEvent[message3.GeneralChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e message3.ChatEvent[message3.GeneralChatBody]) {
		if e.Type != message3.ChatTypeGeneral {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		c, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] chatting.", e.ActorId)
			return
		}

		err = _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), showGeneralChatForSession(l)(ctx)(wp)(e, c.Gm()))
		if err != nil {
			l.WithError(err).Errorf("Unable to send message from character [%d] to map [%d].", e.ActorId, e.MapId)
		}
	}
}

func showGeneralChatForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event message3.ChatEvent[message3.GeneralChatBody], gm bool) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event message3.ChatEvent[message3.GeneralChatBody], gm bool) model.Operator[session.Model] {
		return func(wp writer.Producer) func(event message3.ChatEvent[message3.GeneralChatBody], gm bool) model.Operator[session.Model] {
			return func(event message3.ChatEvent[message3.GeneralChatBody], gm bool) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(chatpkt.GeneralChatWriter)(chatpkt.NewGeneralChat(event.ActorId, gm, event.Message, event.Body.BalloonOnly).Encode)
			}
		}
	}
}

func handleMultiChat(sc server.Model, wp writer.Producer) message.Handler[message3.ChatEvent[message3.MultiChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e message3.ChatEvent[message3.MultiChatBody]) {
		if e.Type == message3.ChatTypeGeneral || e.Type == message3.ChatTypeWhisper {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		// Every channel handler in the world reaches this point; only the
		// channel(s) actually holding a recipient should pay for the sender
		// character GET below.
		present := presentRecipients(l, ctx, sc, e.Body.Recipients)
		if len(present) == 0 {
			return
		}

		c, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] chatting.", e.ActorId)
			return
		}

		for _, cid := range present {
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cid, sendMultiChat(l)(ctx)(wp)(c.Name(), e.Message, message2.MultiChatTypeStrToInd(e.Type)))
			if err != nil {
				l.WithError(err).Errorf("Unable to send message of type [%s] to character [%d].", e.Type, cid)
			}
		}
	}
}

// presentRecipients filters recipients down to those with a session on this
// handler's own channel. Used to avoid paying for a REST lookup (e.g. the
// sender character fetch) on every channel handler in a world when at most
// one channel actually holds any given recipient.
func presentRecipients(l logrus.FieldLogger, ctx context.Context, sc server.Model, recipients []uint32) []uint32 {
	var present []uint32
	for _, cid := range recipients {
		if _, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(cid); err == nil {
			present = append(present, cid)
		}
	}
	return present
}

func sendMultiChat(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(name string, message string, mode byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(name string, message string, mode byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(name string, message string, mode byte) model.Operator[session.Model] {
			return func(name string, message string, mode byte) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(fieldcb.MultiChatWriter)(fieldcb.NewMultiChat(mode, name, message).Encode)
			}
		}
	}
}

// whisperDeliveryPlan decides, for a single (tenant, world, channel) handler
// instance, which of the two whisper announcements it must attempt for one
// whisper chat event.
//
// Whisper is world-scoped in the client, not channel-scoped — WhisperReceive
// carries the sender's channel id as a field precisely so the client can
// render it regardless of which channel the recipient is logged into.
// Handlers are registered once per (tenant, world, channel) socket listener,
// so every channel handler in the event's world must evaluate the recipient
// leg (sendReceive); the session lookup that follows resolves on exactly the
// one channel holding the recipient and is a no-op everywhere else. Only the
// sender's own channel handler emits the send-result confirmation
// (sendResult) — it must be emitted exactly once.
func whisperDeliveryPlan(t tenant.Model, sc server.Model, e message3.ChatEvent[message3.WhisperChatBody]) (sendResult bool, sendReceive bool) {
	if !sc.IsWorld(t, e.WorldId) {
		return false, false
	}
	return sc.Is(t, e.WorldId, e.ChannelId), true
}

func handleWhisperChat(sc server.Model, wp writer.Producer) message.Handler[message3.ChatEvent[message3.WhisperChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e message3.ChatEvent[message3.WhisperChatBody]) {
		if e.Type != message3.ChatTypeWhisper {
			return
		}

		t := tenant.MustFromContext(ctx)
		sendResult, sendReceive := whisperDeliveryPlan(t, sc, e)
		if !sendResult && !sendReceive {
			return
		}

		if sendResult {
			tc, err := character.NewProcessor(l, ctx).GetById()(e.Body.Recipient)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve character [%d] receiving whisper.", e.Body.Recipient)
			} else {
				bp := fieldcb.NewWhisperSendResult(0x0A, tc.Name(), true).Encode
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.ActorId, session.Announce(l)(ctx)(wp)(fieldcb.WhisperWriter)(bp))
				if err != nil {
					l.WithError(err).Errorf("Unable to send whisper message from [%d] to [%d].", e.ActorId, e.Body.Recipient)
				}
			}
		}

		if sendReceive {
			// Test presence before the character fetch: every channel handler
			// in the world reaches this point, and only the one holding the
			// recipient's session should pay for a GET /api/characters/{id}.
			rs, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.Body.Recipient)
			if err == nil {
				c, cErr := character.NewProcessor(l, ctx).GetById()(e.ActorId)
				if cErr != nil {
					l.WithError(cErr).Errorf("Unable to retrieve character [%d] sending whisper.", e.ActorId)
				} else {
					bp := fieldcb.NewWhisperReceive(0x12, c.Name(), byte(e.ChannelId), c.Gm(), e.Message).Encode
					if aErr := session.Announce(l)(ctx)(wp)(fieldcb.WhisperWriter)(bp)(rs); aErr != nil {
						l.WithError(aErr).Errorf("Unable to send whisper message from [%d] to [%d].", e.ActorId, e.Body.Recipient)
					}
				}
			}
		}
	}
}

func handleMessengerChat(sc server.Model, wp writer.Producer) message.Handler[message3.ChatEvent[message3.MessengerChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e message3.ChatEvent[message3.MessengerChatBody]) {
		if e.Type != message3.ChatTypeMessenger {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		for _, cid := range e.Body.Recipients {
			bp := session.Announce(l)(ctx)(wp)(messengercb.MessengerOperationWriter)(messengerpkt.MessengerOperationChatBody(e.Message))
			err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cid, bp)
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

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.Body.OwnerId)
		if err != nil {
			return
		}

		p := pet.NewModelBuilder(e.ActorId, 0, 0, "").SetOwnerID(e.Body.OwnerId).SetSlot(e.Body.PetSlot).MustBuild()
		_ = _map.NewProcessor(l, ctx).ForSessionsInMap(s.Field(), session.Announce(l)(ctx)(wp)(petpkt.PetChatWriter)(petpkt.NewPetChat(p.OwnerId(), p.Slot(), e.Body.Type, e.Body.Action, e.Message, e.Body.Balloon).Encode))
	}
}

func handlePinkChat(sc server.Model, wp writer.Producer) message.Handler[message3.ChatEvent[message3.PinkTextChatBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e message3.ChatEvent[message3.PinkTextChatBody]) {
		if e.Type != message3.ChatTypePinkText {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		// Every channel handler in the world reaches this point; only the
		// channel(s) actually holding a recipient should pay for the sender
		// character GET below.
		present := presentRecipients(l, ctx, sc, e.Body.Recipients)
		if len(present) == 0 {
			return
		}

		characterName := ""

		c, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err == nil {
			characterName = c.Name()
		}

		// TODO retrieve medal name
		for _, cid := range present {
			bp := session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePinkTextBody("", characterName, e.Message))
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cid, bp)
			if err != nil {
				l.WithError(err).Errorf("Unable to send message of type [%s] to character [%d].", e.Type, cid)
			}
		}
	}
}
