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

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("npc_conversation_command")(conversation2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(conversation2.EnvCommandTopic)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSimpleConversationCommand(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleNumberConversationCommand(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStyleConversationCommand(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSlideMenuConversationCommand(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func handleSimpleConversationCommand(sc server.Model, wp writer.Producer) message.Handler[conversation2.CommandEvent[conversation2.CommandSimpleBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c conversation2.CommandEvent[conversation2.CommandSimpleBody]) {
		if c.Type != conversation2.CommandTypeSimple {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), c.WorldId, c.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(c.CharacterId, announceSimpleConversation(l)(ctx)(wp)(c.NpcId, c.Body.Type, c.Message, c.Speaker, c.EndChat, c.SecondaryNpcId))
		if err != nil {
			l.WithError(err).Errorf("Unable to write [%s] for character [%d].", statpkt.StatChangedWriter, c.CharacterId)
		}
	}
}

func handleNumberConversationCommand(sc server.Model, wp writer.Producer) message.Handler[conversation2.CommandEvent[conversation2.CommandNumberBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c conversation2.CommandEvent[conversation2.CommandNumberBody]) {
		if c.Type != conversation2.CommandTypeNumber {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), c.WorldId, c.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(c.CharacterId, announceNumberConversation(l)(ctx)(wp)(c.NpcId, "NUM", c.Message, c.Body.DefaultValue, c.Body.MinValue, c.Body.MaxValue, c.Speaker, c.EndChat, c.SecondaryNpcId))
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

		if !sc.Is(tenant.MustFromContext(ctx), c.WorldId, c.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(c.CharacterId, announceStyleConversation(l)(ctx)(wp)(c.NpcId, "STYLE", c.Message, c.Body.Styles, c.Speaker, c.EndChat, c.SecondaryNpcId))
		if err != nil {
			l.WithError(err).Errorf("Unable to write style conversation for character [%d].", c.CharacterId)
		}
	}
}

func handleSlideMenuConversationCommand(sc server.Model, wp writer.Producer) message.Handler[conversation2.CommandEvent[conversation2.CommandSlideMenuBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c conversation2.CommandEvent[conversation2.CommandSlideMenuBody]) {
		if c.Type != conversation2.CommandTypeSlideMenu {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), c.WorldId, c.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(c.CharacterId, announceSlideMenuConversation(l)(ctx)(wp)(c.NpcId, c.Message, c.Body.MenuType, c.Speaker, c.EndChat, c.SecondaryNpcId))
		if err != nil {
			l.WithError(err).Errorf("Unable to write slide menu conversation for character [%d].", c.CharacterId)
		}
	}
}

func announceSimpleConversation(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType string, message string, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType string, message string, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(npcId uint32, talkType string, message string, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
			return func(npcId uint32, talkType string, message string, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
				scm := &npcpkt.SayConversationDetail{Message: message}
				if talkType == "NEXT" || talkType == "NEXT_PREVIOUS" {
					scm.Next = true
				}
				if talkType == "PREVIOUS" || talkType == "NEXT_PREVIOUS" {
					scm.Previous = true
				}
				speakerByte := computeSpeakerByte(speaker, endChat, secondaryNpcId)
				ncm := model2.NewNpcConversation(npcId, getNPCTalkType(talkType), speakerByte, secondaryNpcId, scm)

				return session.Announce(l)(ctx)(wp)(npcpkt.NpcConversationWriter)(ncm.Encoder)
			}
		}
	}
}

func announceNumberConversation(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType string, message string, def uint32, min uint32, max uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType string, message string, def uint32, min uint32, max uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(npcId uint32, talkType string, message string, def uint32, min uint32, max uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
			return func(npcId uint32, talkType string, message string, def uint32, min uint32, max uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
				scm := &npcpkt.AskNumberConversationDetail{Message: message, Def: def, Min: min, Max: max}
				speakerByte := computeSpeakerByte(speaker, endChat, secondaryNpcId)
				ncm := model2.NewNpcConversation(npcId, getNPCTalkType(talkType), speakerByte, secondaryNpcId, scm)
				return session.Announce(l)(ctx)(wp)(npcpkt.NpcConversationWriter)(ncm.Encoder)
			}
		}
	}
}

func announceStyleConversation(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType string, message string, styles []uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType string, message string, styles []uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(npcId uint32, talkType string, message string, styles []uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
			return func(npcId uint32, talkType string, message string, styles []uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
				scm := &npcpkt.AskAvatarConversationDetail{Message: message, Styles: styles}
				speakerByte := computeSpeakerByte(speaker, endChat, secondaryNpcId)
				ncm := model2.NewNpcConversation(npcId, getNPCTalkType(talkType), speakerByte, secondaryNpcId, scm)
				return session.Announce(l)(ctx)(wp)(npcpkt.NpcConversationWriter)(ncm.Encoder)
			}
		}
	}
}

func announceSlideMenuConversation(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, message string, menuType uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, message string, menuType uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(npcId uint32, message string, menuType uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
			return func(npcId uint32, message string, menuType uint32, speaker string, endChat bool, secondaryNpcId uint32) model.Operator[session.Model] {
				scm := &npcpkt.AskSlideMenuConversationDetail{Message: message, MenuType: menuType}
				speakerByte := computeSpeakerByte(speaker, endChat, secondaryNpcId)
				ncm := model2.NewNpcConversation(npcId, npcpkt.NpcConversationMessageTypeAskSlideMenu, speakerByte, secondaryNpcId, scm)
				return session.Announce(l)(ctx)(wp)(npcpkt.NpcConversationWriter)(ncm.Encoder)
			}
		}
	}
}

// computeSpeakerByte calculates the speaker byte for the client protocol.
// Bit 0: end chat visibility (0 = show, 1 = hide)
// Bit 1: speaker type (0 = NPC, 1 = CHARACTER)
// Bit 2: secondary NPC (0 = none, 1 = has secondary NPC template ID)
func computeSpeakerByte(speaker string, endChat bool, secondaryNpcId uint32) byte {
	var b byte = 0
	if !endChat {
		b |= 1
	}
	if speaker == "CHARACTER" {
		b |= 2
	}
	if secondaryNpcId != 0 {
		b |= 4
	}
	return b
}

func getNPCTalkType(t string) npcpkt.NpcConversationMessageType {
	switch t {
	case "NEXT":
		return npcpkt.NpcConversationMessageTypeSay
	case "PREVIOUS":
		return npcpkt.NpcConversationMessageTypeSay
	case "NEXT_PREVIOUS":
		return npcpkt.NpcConversationMessageTypeSay
	case "OK":
		return npcpkt.NpcConversationMessageTypeSay
	case "YES_NO":
		return npcpkt.NpcConversationMessageTypeAskYesNo
	case "NUM":
		return npcpkt.NpcConversationMessageTypeAskNumber
	case "SIMPLE":
		return npcpkt.NpcConversationMessageTypeAskMenu
	case "STYLE":
		return npcpkt.NpcConversationMessageTypeAskAvatar
	case "ACCEPT_DECLINE":
		return npcpkt.NpcConversationMessageTypeAskYesNoQuest
	}
	panic(fmt.Sprintf("unsupported talk type %s", t))
}
