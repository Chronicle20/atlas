package quest

import (
	"atlas-channel/kafka/message/quest"
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	StartQuestConversation(m _map.Model, questId uint32, npcId uint32, characterId uint32) error
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

func (p *ProcessorImpl) StartQuestConversation(m _map.Model, questId uint32, npcId uint32, characterId uint32) error {
	p.l.Debugf("Starting quest [%d] conversation for character [%d] with NPC [%d].", questId, characterId, npcId)
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(StartConversationCommandProvider(m, questId, npcId, characterId))
}
