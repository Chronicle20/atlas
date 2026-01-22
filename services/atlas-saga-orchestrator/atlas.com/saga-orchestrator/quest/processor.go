package quest

import (
	"atlas-saga-orchestrator/kafka/message/quest"
	"atlas-saga-orchestrator/kafka/producer"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	RequestStartQuest(transactionId uuid.UUID, worldId byte, characterId uint32, questId uint32, npcId uint32) error
	RequestCompleteQuest(transactionId uuid.UUID, worldId byte, characterId uint32, questId uint32, npcId uint32, selection int32, force bool) error
	RequestForfeitQuest(transactionId uuid.UUID, worldId byte, characterId uint32, questId uint32) error
	RequestUpdateProgress(transactionId uuid.UUID, worldId byte, characterId uint32, questId uint32, infoNumber uint32, progress string) error
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

func (p *ProcessorImpl) RequestStartQuest(transactionId uuid.UUID, worldId byte, characterId uint32, questId uint32, npcId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(StartQuestCommandProvider(transactionId, worldId, characterId, questId, npcId))
}

func (p *ProcessorImpl) RequestCompleteQuest(transactionId uuid.UUID, worldId byte, characterId uint32, questId uint32, npcId uint32, selection int32, force bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(CompleteQuestCommandProvider(transactionId, worldId, characterId, questId, npcId, selection, force))
}

func (p *ProcessorImpl) RequestForfeitQuest(transactionId uuid.UUID, worldId byte, characterId uint32, questId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(ForfeitQuestCommandProvider(transactionId, worldId, characterId, questId))
}

func (p *ProcessorImpl) RequestUpdateProgress(transactionId uuid.UUID, worldId byte, characterId uint32, questId uint32, infoNumber uint32, progress string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(UpdateProgressCommandProvider(transactionId, worldId, characterId, questId, infoNumber, progress))
}
