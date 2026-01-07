package quest

import (
	"atlas-saga-orchestrator/kafka/message/quest"
	"atlas-saga-orchestrator/kafka/producer"
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	RequestStartQuest(worldId byte, characterId uint32, questId uint32, npcId uint32) error
	RequestCompleteQuest(worldId byte, characterId uint32, questId uint32, npcId uint32, selection int32, force bool) error
	RequestForfeitQuest(worldId byte, characterId uint32, questId uint32) error
	RequestUpdateProgress(worldId byte, characterId uint32, questId uint32, infoNumber uint32, progress string) error
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

func (p *ProcessorImpl) RequestStartQuest(worldId byte, characterId uint32, questId uint32, npcId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(StartQuestCommandProvider(worldId, characterId, questId, npcId))
}

func (p *ProcessorImpl) RequestCompleteQuest(worldId byte, characterId uint32, questId uint32, npcId uint32, selection int32, force bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(CompleteQuestCommandProvider(worldId, characterId, questId, npcId, selection, force))
}

func (p *ProcessorImpl) RequestForfeitQuest(worldId byte, characterId uint32, questId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(ForfeitQuestCommandProvider(worldId, characterId, questId))
}

func (p *ProcessorImpl) RequestUpdateProgress(worldId byte, characterId uint32, questId uint32, infoNumber uint32, progress string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(UpdateProgressCommandProvider(worldId, characterId, questId, infoNumber, progress))
}
