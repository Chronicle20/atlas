package quest

import (
	"atlas-saga-orchestrator/kafka/message/quest"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Processor interface {
	RequestStartQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, npcId uint32, rewards []quest.ItemReward) error
	RequestCompleteQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, npcId uint32, selection int32, force bool, rewards []quest.ItemReward) error
	RequestForfeitQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32) error
	RequestUpdateProgress(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, infoNumber uint32, progress string) error

	// RequestExplorerQuest credits one exploration region for a medal-style
	// quest (task-290 G14): force-start the quest via npc 9000066, then
	// synchronously record the map on atlas-quest's medal-map set, which
	// itself performs the dedup Cosmic does with
	// `if (!qs.addMedalMap(...)) return;` (MapScriptMethods.java:104-139).
	// It does not write quest progress or send a completion/progress
	// message -- see ExplorerQuestPayload's doc comment in libs/atlas-saga
	// for why.
	RequestExplorerQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, mapId uint32) error
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) RequestStartQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, npcId uint32, rewards []quest.ItemReward) error {
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(StartQuestCommandProvider(transactionId, worldId, characterId, questId, npcId, rewards))
}

func (p *ProcessorImpl) RequestCompleteQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, npcId uint32, selection int32, force bool, rewards []quest.ItemReward) error {
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(CompleteQuestCommandProvider(transactionId, worldId, characterId, questId, npcId, selection, force, rewards))
}

func (p *ProcessorImpl) RequestForfeitQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(ForfeitQuestCommandProvider(transactionId, worldId, characterId, questId))
}

func (p *ProcessorImpl) RequestUpdateProgress(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, infoNumber uint32, progress string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(UpdateProgressCommandProvider(transactionId, worldId, characterId, questId, infoNumber, progress))
}

// explorerQuestNpcId is Cosmic's fixed force-start NPC for explorerQuest
// (MapScriptMethods.java:104-139: quest.forceStart(getPlayer(), 9000066)).
const explorerQuestNpcId uint32 = 9000066

func (p *ProcessorImpl) RequestExplorerQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, mapId uint32) error {
	if err := p.RequestStartQuest(transactionId, worldId, characterId, questId, explorerQuestNpcId, nil); err != nil {
		p.l.WithError(err).Errorf("Unable to force-start quest [%d] for character [%d] as part of explorer_quest.", questId, characterId)
		return err
	}

	result, err := postMedalMap(p.l, p.ctx)(characterId, questId, mapId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to record medal map [%d] for character [%d] quest [%d].", mapId, characterId, questId)
		return err
	}

	p.l.Debugf("Recorded medal map [%d] for character [%d] quest [%d]; count now [%d].", mapId, characterId, questId, result.Count)
	return nil
}
