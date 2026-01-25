package npc

import (
	npc2 "atlas-npc-conversations/kafka/message/npc"
	"atlas-npc-conversations/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
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

	SpeakerNPC       = "NPC"
	SpeakerCharacter = "CHARACTER"
)

type Processor interface {
	Dispose(worldId world.Id, channelId channel.Id, characterId uint32)
	SendSimple(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc
	SendNext(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc
	SendNextPrevious(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc
	SendPrevious(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc
	SendOk(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc
	SendYesNo(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc
	SendAcceptDecline(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc
	SendNumber(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32, message string, def uint32, min uint32, max uint32) error
	SendStyle(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32, message string, styles []uint32) error
	SendNPCTalk(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32, config *TalkConfig) func(message string, configurations ...TalkConfigurator)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) Dispose(worldId world.Id, channelId channel.Id, characterId uint32) {
	_ = producer.ProviderImpl(p.l)(p.ctx)(npc2.EnvEventTopicCharacterStatus)(enableActionsProvider(worldId, channelId, characterId))
}

type TalkConfig struct {
	messageType    string
	speaker        string
	endChat        bool
	secondaryNpcId uint32
}

func (c TalkConfig) MessageType() string {
	return c.messageType
}

func (c TalkConfig) Speaker() string {
	return c.speaker
}

func (c TalkConfig) EndChat() bool {
	return c.endChat
}

func (c TalkConfig) SecondaryNpcId() uint32 {
	return c.secondaryNpcId
}

type TalkConfigurator func(config *TalkConfig)

type TalkFunc func(message string, configurations ...TalkConfigurator)

// WithSpeaker returns a TalkConfigurator that sets the speaker
func WithSpeaker(speaker string) TalkConfigurator {
	return func(config *TalkConfig) {
		if speaker != "" {
			config.speaker = speaker
		}
	}
}

// WithEndChat returns a TalkConfigurator that sets whether to show the end chat button
func WithEndChat(endChat bool) TalkConfigurator {
	return func(config *TalkConfig) {
		config.endChat = endChat
	}
}

// WithSecondaryNpcId returns a TalkConfigurator that sets the secondary NPC template ID
func WithSecondaryNpcId(npcId uint32) TalkConfigurator {
	return func(config *TalkConfig) {
		config.secondaryNpcId = npcId
	}
}

func (p *ProcessorImpl) SendSimple(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(worldId, channelId, characterId, npcId, &TalkConfig{messageType: MessageTypeSimple, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendNext(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(worldId, channelId, characterId, npcId, &TalkConfig{messageType: MessageTypeNext, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendNextPrevious(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(worldId, channelId, characterId, npcId, &TalkConfig{messageType: MessageTypeNextPrevious, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendPrevious(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(worldId, channelId, characterId, npcId, &TalkConfig{messageType: MessageTypePrevious, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendOk(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(worldId, channelId, characterId, npcId, &TalkConfig{messageType: MessageTypeOk, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendYesNo(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(worldId, channelId, characterId, npcId, &TalkConfig{messageType: MessageTypeYesNo, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendAcceptDecline(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(worldId, channelId, characterId, npcId, &TalkConfig{messageType: MessageTypeAcceptDecline, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendNumber(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32, message string, def uint32, min uint32, max uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(npc2.EnvConversationCommandTopic)(numberConversationProvider(worldId, channelId, characterId, npcId, message, def, min, max, SpeakerNPC, true, 0))
}

func (p *ProcessorImpl) SendStyle(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32, message string, styles []uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(npc2.EnvConversationCommandTopic)(styleConversationProvider(worldId, channelId, characterId, npcId, message, styles, SpeakerNPC, true, 0))
}

func (p *ProcessorImpl) SendNPCTalk(worldId world.Id, channelId channel.Id, characterId uint32, npcId uint32, config *TalkConfig) func(message string, configurations ...TalkConfigurator) {
	return func(message string, configurations ...TalkConfigurator) {
		for _, configuration := range configurations {
			configuration(config)
		}
		_ = producer.ProviderImpl(p.l)(p.ctx)(npc2.EnvConversationCommandTopic)(simpleConversationProvider(worldId, channelId, characterId, npcId, message, config.MessageType(), config.Speaker(), config.EndChat(), config.SecondaryNpcId()))
	}
}
