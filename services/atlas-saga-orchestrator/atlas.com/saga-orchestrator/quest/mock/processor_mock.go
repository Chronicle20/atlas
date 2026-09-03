package mock

import (
	"atlas-saga-orchestrator/kafka/message/quest"
	questp "atlas-saga-orchestrator/quest"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// ProcessorMock is a mock implementation of the quest.Processor interface.
type ProcessorMock struct {
	RequestStartQuestFunc     func(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, npcId uint32, rewards []quest.ItemReward) error
	RequestCompleteQuestFunc  func(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, npcId uint32, selection int32, force bool, rewards []quest.ItemReward) error
	RequestForfeitQuestFunc   func(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32) error
	RequestUpdateProgressFunc func(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, infoNumber uint32, progress string) error
	RequestExplorerQuestFunc  func(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, mapId uint32) error
}

var _ questp.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) RequestStartQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, npcId uint32, rewards []quest.ItemReward) error {
	if m.RequestStartQuestFunc != nil {
		return m.RequestStartQuestFunc(transactionId, worldId, characterId, questId, npcId, rewards)
	}
	return nil
}

func (m *ProcessorMock) RequestCompleteQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, npcId uint32, selection int32, force bool, rewards []quest.ItemReward) error {
	if m.RequestCompleteQuestFunc != nil {
		return m.RequestCompleteQuestFunc(transactionId, worldId, characterId, questId, npcId, selection, force, rewards)
	}
	return nil
}

func (m *ProcessorMock) RequestForfeitQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32) error {
	if m.RequestForfeitQuestFunc != nil {
		return m.RequestForfeitQuestFunc(transactionId, worldId, characterId, questId)
	}
	return nil
}

func (m *ProcessorMock) RequestUpdateProgress(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, infoNumber uint32, progress string) error {
	if m.RequestUpdateProgressFunc != nil {
		return m.RequestUpdateProgressFunc(transactionId, worldId, characterId, questId, infoNumber, progress)
	}
	return nil
}

func (m *ProcessorMock) RequestExplorerQuest(transactionId uuid.UUID, worldId world.Id, characterId uint32, questId uint32, mapId uint32) error {
	if m.RequestExplorerQuestFunc != nil {
		return m.RequestExplorerQuestFunc(transactionId, worldId, characterId, questId, mapId)
	}
	return nil
}
