package quest

import (
	"atlas-saga-orchestrator/kafka/message/quest"
	"context"
	"strconv"

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
	// quest (task-290 G14/C22b): force-start the quest via npc 9000066, then
	// synchronously record the map on atlas-quest's medal-map set, which
	// itself performs the dedup Cosmic does with
	// `if (!qs.addMedalMap(...)) return;` (MapScriptMethods.java:104-139).
	// When the map is newly recorded, it also resolves the quest's
	// infoNumber/infoEx threshold from atlas-data and returns them so the
	// caller can write quest progress the same way handleSetQuestProgress
	// does. It does not send the completion/progress client packet -- see
	// ExplorerQuestResult's doc comment for why.
	RequestExplorerQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, mapId uint32) (ExplorerQuestResult, error)
}

// ExplorerQuestResult is RequestExplorerQuest's outcome. Count and
// NewlyRecorded come straight from atlas-quest's medal-map record; InfoNumber
// and Threshold are resolved from atlas-data only when NewlyRecorded is true
// (Cosmic never reads them on the dedup-rejected path either).
//
// Cosmic's explorerQuest then sends either getShowQuestCompletion(questId)
// or earnTitleMessage("<n>/<m> regions explored.") depending on the
// Count-vs-Threshold comparison (MapScriptMethods.java:128-136). Neither is
// sent here: both are client packets, and no existing writer/action in this
// service covers them -- sending them would require a new packet codec,
// which is out of scope for this task (see task-C22b-brief.md's scope
// boundary). The caller logs the comparison instead.
type ExplorerQuestResult struct {
	Count         uint32
	NewlyRecorded bool
	InfoNumber    uint32
	Threshold     int
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

func (p *ProcessorImpl) RequestExplorerQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, mapId uint32) (ExplorerQuestResult, error) {
	if err := p.RequestStartQuest(transactionId, worldId, characterId, questId, explorerQuestNpcId, nil); err != nil {
		p.l.WithError(err).Errorf("Unable to force-start quest [%d] for character [%d] as part of explorer_quest.", questId, characterId)
		return ExplorerQuestResult{}, err
	}

	medalResult, err := postMedalMap(p.l, p.ctx)(characterId, questId, mapId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to record medal map [%d] for character [%d] quest [%d].", mapId, characterId, questId)
		return ExplorerQuestResult{}, err
	}

	if !medalResult.NewlyRecorded {
		p.l.Debugf("Medal map [%d] already recorded for character [%d] quest [%d]; count [%d]; skipping progress write.", mapId, characterId, questId, medalResult.Count)
		return ExplorerQuestResult{Count: medalResult.Count, NewlyRecorded: false}, nil
	}
	p.l.Debugf("Recorded medal map [%d] for character [%d] quest [%d]; count now [%d].", mapId, characterId, questId, medalResult.Count)

	// The quest was just force-started above, so its status is STARTED;
	// Quest.getInfoNumber(Status)/getInfoEx(Status, index) (Quest.java:462-485)
	// both read END requirements for that status.
	questData, err := requestQuestData(p.ctx, questId)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to load quest [%d] definition to resolve explorer_quest infoNumber/infoEx.", questId)
		return ExplorerQuestResult{}, err
	}

	infoNumber := questData.EndRequirements.InfoNumber
	if infoNumber == 0 {
		// Cosmic's fallback: if (infoNumber <= 0) infoNumber = questid; (Quest.java:268-270).
		infoNumber = questId
	}

	threshold := 0
	if len(questData.EndRequirements.InfoEx) > 0 {
		if t, tErr := strconv.Atoi(questData.EndRequirements.InfoEx[0]); tErr == nil {
			threshold = t
		}
	}

	return ExplorerQuestResult{
		Count:         medalResult.Count,
		NewlyRecorded: true,
		InfoNumber:    infoNumber,
		Threshold:     threshold,
	}, nil
}
