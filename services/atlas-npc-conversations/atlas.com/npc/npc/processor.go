package npc

import (
	npc2 "atlas-npc-conversations/kafka/message/npc"
	"atlas-npc-conversations/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
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
	Dispose(ch channel.Model, characterId uint32)
	SendSimple(ch channel.Model, characterId uint32, npcId uint32) TalkFunc
	SendNext(ch channel.Model, characterId uint32, npcId uint32) TalkFunc
	SendNextPrevious(ch channel.Model, characterId uint32, npcId uint32) TalkFunc
	SendPrevious(ch channel.Model, characterId uint32, npcId uint32) TalkFunc
	SendOk(ch channel.Model, characterId uint32, npcId uint32) TalkFunc
	SendYesNo(ch channel.Model, characterId uint32, npcId uint32) TalkFunc
	SendAcceptDecline(ch channel.Model, characterId uint32, npcId uint32) TalkFunc
	SendNumber(ch channel.Model, characterId uint32, npcId uint32, message string, def uint32, min uint32, max uint32) error
	SendStyle(ch channel.Model, characterId uint32, npcId uint32, message string, styles []uint32) error
	SendNPCTalk(ch channel.Model, characterId uint32, npcId uint32, config *TalkConfig) func(message string, configurations ...TalkConfigurator)
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

func (p *ProcessorImpl) Dispose(ch channel.Model, characterId uint32) {
	_ = producer.ProviderImpl(p.l)(p.ctx)(npc2.EnvEventTopicCharacterStatus)(enableActionsProvider(ch, characterId))
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

func (p *ProcessorImpl) SendSimple(ch channel.Model, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(ch, characterId, npcId, &TalkConfig{messageType: MessageTypeSimple, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendNext(ch channel.Model, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(ch, characterId, npcId, &TalkConfig{messageType: MessageTypeNext, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendNextPrevious(ch channel.Model, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(ch, characterId, npcId, &TalkConfig{messageType: MessageTypeNextPrevious, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendPrevious(ch channel.Model, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(ch, characterId, npcId, &TalkConfig{messageType: MessageTypePrevious, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendOk(ch channel.Model, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(ch, characterId, npcId, &TalkConfig{messageType: MessageTypeOk, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendYesNo(ch channel.Model, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(ch, characterId, npcId, &TalkConfig{messageType: MessageTypeYesNo, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendAcceptDecline(ch channel.Model, characterId uint32, npcId uint32) TalkFunc {
	return p.SendNPCTalk(ch, characterId, npcId, &TalkConfig{messageType: MessageTypeAcceptDecline, speaker: SpeakerNPC, endChat: true})
}

func (p *ProcessorImpl) SendNumber(ch channel.Model, characterId uint32, npcId uint32, message string, def uint32, min uint32, max uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(npc2.EnvConversationCommandTopic)(numberConversationProvider(ch, characterId, npcId, message, def, min, max, SpeakerNPC, true, 0))
}

func (p *ProcessorImpl) SendStyle(ch channel.Model, characterId uint32, npcId uint32, message string, styles []uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(npc2.EnvConversationCommandTopic)(styleConversationProvider(ch, characterId, npcId, message, styles, SpeakerNPC, true, 0))
}

func (p *ProcessorImpl) SendNPCTalk(ch channel.Model, characterId uint32, npcId uint32, config *TalkConfig) func(message string, configurations ...TalkConfigurator) {
	return func(message string, configurations ...TalkConfigurator) {
		for _, configuration := range configurations {
			configuration(config)
		}
		_ = producer.ProviderImpl(p.l)(p.ctx)(npc2.EnvConversationCommandTopic)(simpleConversationProvider(ch, characterId, npcId, message, config.MessageType(), config.Speaker(), config.EndChat(), config.SecondaryNpcId()))
	}
}
