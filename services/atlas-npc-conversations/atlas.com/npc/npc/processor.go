package npc

import (
	npc2 "atlas-npc-conversations/kafka/message/npc"
	"atlas-npc-conversations/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

const (
	MessageTypeSimple        = "SIMPLE"
	MessageTypeNext          = "NEXT"
	MessageTypeNextPrevious  = "NEXT_PREVIOUS"
	MessageTypePrevious      = "PREVIOUS"
	MessageTypeYesNo         = "YES_NO"
	MessageTypeOk            = "OK"
	MessageTypeNum           = "NUM"
	MessageTypeText          = "TEXT"
	MessageTypeStyle         = "STYLE"
	MessageTypeAcceptDecline = "ACCEPT_DECLINE"

	SpeakerNPCLeft        = "NPC_LEFT"
	SpeakerNPCRight       = "NPC_RIGHT"
	SpeakerCharacterLeft  = "CHARACTER_LEFT"
	SpeakerCharacterRight = "CHARACTER_RIGHT"
	SpeakerUnknown        = "UNKNOWN"
	SpeakerUnknown2       = "UNKNOWN2"
)

func Dispose(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) {
		return func(worldId byte, channelId byte, characterId uint32) {
			_ = producer.ProviderImpl(l)(ctx)(npc2.EnvEventTopicCharacterStatus)(enableActionsProvider(worldId, channelId, characterId))
		}
	}
}

type TalkConfig struct {
	messageType string
	speaker     string
}

func (c TalkConfig) MessageType() string {
	return c.messageType
}

func (c TalkConfig) Speaker() string {
	return c.speaker
}

type TalkConfigurator func(config *TalkConfig)

type TalkFunc func(message string, configurations ...TalkConfigurator)

func SendSimple(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
		return func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
			return SendNPCTalk(l)(ctx)(worldId, channelId, characterId, npcId, &TalkConfig{messageType: MessageTypeSimple, speaker: SpeakerNPCLeft})
		}
	}
}

func SendNext(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
		return func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
			return SendNPCTalk(l)(ctx)(worldId, channelId, characterId, npcId, &TalkConfig{messageType: MessageTypeNext, speaker: SpeakerNPCLeft})
		}
	}
}

func SendNextPrevious(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
		return func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
			return SendNPCTalk(l)(ctx)(worldId, channelId, characterId, npcId, &TalkConfig{messageType: MessageTypeNextPrevious, speaker: SpeakerNPCLeft})
		}
	}
}

func SendOk(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
		return func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
			return SendNPCTalk(l)(ctx)(worldId, channelId, characterId, npcId, &TalkConfig{messageType: MessageTypeOk, speaker: SpeakerNPCLeft})
		}
	}
}

func SendYesNo(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
		return func(worldId byte, channelId byte, characterId uint32, npcId uint32) TalkFunc {
			return SendNPCTalk(l)(ctx)(worldId, channelId, characterId, npcId, &TalkConfig{messageType: MessageTypeYesNo, speaker: SpeakerNPCLeft})
		}
	}
}

func SendNPCTalk(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, npcId uint32, config *TalkConfig) func(message string, configurations ...TalkConfigurator) {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, npcId uint32, config *TalkConfig) func(message string, configurations ...TalkConfigurator) {
		return func(worldId byte, channelId byte, characterId uint32, npcId uint32, config *TalkConfig) func(message string, configurations ...TalkConfigurator) {
			return func(message string, configurations ...TalkConfigurator) {
				for _, configuration := range configurations {
					configuration(config)
				}
				_ = producer.ProviderImpl(l)(ctx)(npc2.EnvConversationCommandTopic)(simpleConversationProvider(worldId, channelId, characterId, npcId, message, config.MessageType(), config.Speaker()))
			}
		}
	}
}
