package conversation

import (
	consumer2 "atlas-channel/kafka/consumer"
	conversation2 "atlas-channel/kafka/message/npc/conversation"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
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
			rf(consumer2.NewConfig(l)("npc_conversation_command")(conversation2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(conversation2.EnvCommandTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSimpleConversationCommand(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleNumberConversationCommand(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStyleConversationCommand(sc, wp))))
			}
		}
	}
}

func handleSimpleConversationCommand(sc server.Model, wp writer.Producer) message.Handler[conversation2.CommandEvent[conversation2.CommandSimpleBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c conversation2.CommandEvent[conversation2.CommandSimpleBody]) {
		if c.Type != conversation2.CommandTypeSimple {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(c.WorldId), channel.Id(c.ChannelId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(c.CharacterId, announceSimpleConversation(l)(ctx)(wp)(c.NpcId, c.Body.Type, c.Message, c.Speaker))
		if err != nil {
			l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, c.CharacterId)
		}
	}
}

func handleNumberConversationCommand(sc server.Model, wp writer.Producer) message.Handler[conversation2.CommandEvent[conversation2.CommandNumberBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c conversation2.CommandEvent[conversation2.CommandNumberBody]) {
		if c.Type != conversation2.CommandTypeNumber {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(c.WorldId), channel.Id(c.ChannelId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(c.CharacterId, announceNumberConversation(l)(ctx)(wp)(c.NpcId, "NUM", c.Message, c.Body.DefaultValue, c.Body.MinValue, c.Body.MaxValue, getNPCTalkSpeaker(c.Speaker)))
		if err != nil {
			l.WithError(err).Errorf("Unable to write number conversation for character [%d].", c.CharacterId)
		}
	}
}

func handleStyleConversationCommand(sc server.Model, wp writer.Producer) message.Handler[conversation2.CommandEvent[conversation2.CommandStyleBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c conversation2.CommandEvent[conversation2.CommandStyleBody]) {
		if c.Type != conversation2.CommandTypeStyle {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(c.WorldId), channel.Id(c.ChannelId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(c.CharacterId, announceStyleConversation(l)(ctx)(wp)(c.NpcId, "STYLE", c.Message, c.Body.Styles, getNPCTalkSpeaker(c.Speaker)))
		if err != nil {
			l.WithError(err).Errorf("Unable to write style conversation for character [%d].", c.CharacterId)
		}
	}
}

func announceSimpleConversation(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType string, message string, speaker string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType string, message string, speaker string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(npcId uint32, talkType string, message string, speaker string) model.Operator[session.Model] {
			return func(npcId uint32, talkType string, message string, speaker string) model.Operator[session.Model] {
				t := tenant.MustFromContext(ctx)
				scm := &model2.SayConversationDetail{Message: message}
				if talkType == "NEXT" || talkType == "NEXT_PREVIOUS" {
					scm.Next = true
				}
				if talkType == "PREVIOUS" || talkType == "NEXT_PREVIOUS" {
					scm.Previous = true
				}
				ncm := model2.NewNpcConversation(npcId, getNPCTalkType(talkType), getNPCTalkSpeaker(speaker), scm)

				return session.Announce(l)(ctx)(wp)(writer.NPCConversation)(writer.NPCConversationBody(l, t)(ncm))
			}
		}
	}
}

func announceNumberConversation(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType string, message string, def uint32, min uint32, max uint32, speaker byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType string, message string, def uint32, min uint32, max uint32, speaker byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(npcId uint32, talkType string, message string, def uint32, min uint32, max uint32, speaker byte) model.Operator[session.Model] {
			return func(npcId uint32, talkType string, message string, def uint32, min uint32, max uint32, speaker byte) model.Operator[session.Model] {
				t := tenant.MustFromContext(ctx)
				scm := &model2.AskNumberConversationDetail{Message: message, Def: def, Min: min, Max: max}
				ncm := model2.NewNpcConversation(npcId, getNPCTalkType(talkType), 0, scm)
				return session.Announce(l)(ctx)(wp)(writer.NPCConversation)(writer.NPCConversationBody(l, t)(ncm))
			}
		}
	}
}

func announceStyleConversation(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType string, message string, styles []uint32, speaker byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType string, message string, styles []uint32, speaker byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(npcId uint32, talkType string, message string, styles []uint32, speaker byte) model.Operator[session.Model] {
			return func(npcId uint32, talkType string, message string, styles []uint32, speaker byte) model.Operator[session.Model] {
				t := tenant.MustFromContext(ctx)
				scm := &model2.AskAvatarConversationDetail{Message: message, Styles: styles}
				ncm := model2.NewNpcConversation(npcId, getNPCTalkType(talkType), 0, scm)
				return session.Announce(l)(ctx)(wp)(writer.NPCConversation)(writer.NPCConversationBody(l, t)(ncm))
			}
		}
	}
}

func getNPCTalkSpeaker(speaker string) byte {
	switch speaker {
	case "NPC_LEFT":
		return 0
	case "NPC_RIGHT":
		return 1
	case "CHARACTER_LEFT":
		return 2
	case "CHARACTER_RIGHT":
		return 3
	}
	panic(fmt.Sprintf("unsupported npc talk speaker %s", speaker))
}

func getNPCTalkType(t string) model2.NpcConversationMessageType {
	switch t {
	case "NEXT":
		return model2.NpcConversationMessageTypeSay
	case "PREVIOUS":
		return model2.NpcConversationMessageTypeSay
	case "NEXT_PREVIOUS":
		return model2.NpcConversationMessageTypeSay
	case "OK":
		return model2.NpcConversationMessageTypeSay
	case "YES_NO":
		return model2.NpcConversationMessageTypeAskYesNo
	case "ACCEPT_DECLINE":
		return model2.NpcConversationMessageTypeAskYesNo
	case "SIMPLE":
		return model2.NpcConversationMessageTypeSay
	case "NUM":
		return model2.NpcConversationMessageTypeAskNumber
	case "STYLE":
		return model2.NpcConversationMessageTypeAskAvatar
	}
	panic(fmt.Sprintf("unsupported talk type %s", t))
}
