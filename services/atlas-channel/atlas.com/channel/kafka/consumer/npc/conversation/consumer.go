package conversation

import (
	consumer2 "atlas-channel/kafka/consumer"
	conversation2 "atlas-channel/kafka/message/npc/conversation"
	"atlas-channel/server"
	"atlas-channel/session"
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

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(c.CharacterId, announceSimpleConversation(l)(ctx)(wp)(c.NpcId, getNPCTalkType(c.Body.Type), c.Message, getNPCTalkEnd(c.Body.Type), getNPCTalkSpeaker(c.Speaker)))
		if err != nil {
			l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, c.CharacterId)
		}
	}
}

func announceSimpleConversation(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType byte, message string, endType []byte, speaker byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(npcId uint32, talkType byte, message string, endType []byte, speaker byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(npcId uint32, talkType byte, message string, endType []byte, speaker byte) model.Operator[session.Model] {
			return func(npcId uint32, talkType byte, message string, endType []byte, speaker byte) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.NPCConversation)(writer.NPCConversationBody(l)(npcId, talkType, message, endType, speaker))
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

func getNPCTalkEnd(t string) []byte {
	switch t {
	case "NEXT":
		return []byte{00, 01}
	case "PREVIOUS":
		return []byte{01, 00}
	case "NEXT_PREVIOUS":
		return []byte{01, 01}
	case "OK":
		return []byte{00, 00}
	case "YES_NO":
		return []byte{}
	case "ACCEPT_DECLINE":
		return []byte{}
	case "SIMPLE":
		return []byte{}
	}
	panic(fmt.Sprintf("unsupported talk type %s", t))
}

func getNPCTalkType(t string) byte {
	switch t {
	case "NEXT":
		return 0
	case "PREVIOUS":
		return 0
	case "NEXT_PREVIOUS":
		return 0
	case "OK":
		return 0
	case "YES_NO":
		return 1
	case "ACCEPT_DECLINE":
		return 0x0C
	case "SIMPLE":
		return 4
	}
	panic(fmt.Sprintf("unsupported talk type %s", t))
}
