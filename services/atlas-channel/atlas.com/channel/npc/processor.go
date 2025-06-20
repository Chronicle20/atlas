package npc

import (
	"atlas-channel/kafka/message/npc"
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	StartConversation(m _map.Model, npcId uint32, characterId uint32) error
	ContinueConversation(characterId uint32, action byte, lastMessageType byte, selection int32) error
	DisposeConversation(characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *ProcessorImpl) StartConversation(m _map.Model, npcId uint32, characterId uint32) error {
	p.l.Debugf("Starting NPC [%d] conversation for character [%d].", npcId, characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(npc.EnvCommandTopic)(StartConversationCommandProvider(m, npcId, characterId))
}

func (p *ProcessorImpl) ContinueConversation(characterId uint32, action byte, lastMessageType byte, selection int32) error {
	p.l.Debugf("Continuing NPC conversation for character [%d]. action [%d], lastMessageType [%d], selection [%d].", characterId, action, lastMessageType, selection)
	return producer.ProviderImpl(p.l)(p.ctx)(npc.EnvCommandTopic)(ContinueConversationCommandProvider(characterId, action, lastMessageType, selection))
}

func (p *ProcessorImpl) DisposeConversation(characterId uint32) error {
	p.l.Debugf("Ending NPC conversation for character [%d].", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(npc.EnvCommandTopic)(DisposeConversationCommandProvider(characterId))
}
