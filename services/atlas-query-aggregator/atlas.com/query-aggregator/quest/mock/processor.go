package mock

import (
	"atlas-query-aggregator/quest"
	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorImpl is a mock implementation of the quest.Processor interface
type ProcessorImpl struct {
	GetQuestStatusFunc   func(characterId uint32, questId uint32) model.Provider[quest.QuestStatus]
	GetQuestProgressFunc func(characterId uint32, questId uint32, step string) model.Provider[int]
	GetQuestFunc         func(characterId uint32, questId uint32) model.Provider[quest.Model]
}

// GetQuestStatus returns the status of a quest for a character
func (m *ProcessorImpl) GetQuestStatus(characterId uint32, questId uint32) model.Provider[quest.QuestStatus] {
	if m.GetQuestStatusFunc != nil {
		return m.GetQuestStatusFunc(characterId, questId)
	}
	return func() (quest.QuestStatus, error) {
		return quest.UNDEFINED, nil
	}
}

// GetQuestProgress returns the progress of a quest step for a character
func (m *ProcessorImpl) GetQuestProgress(characterId uint32, questId uint32, step string) model.Provider[int] {
	if m.GetQuestProgressFunc != nil {
		return m.GetQuestProgressFunc(characterId, questId, step)
	}
	return func() (int, error) {
		return 0, nil
	}
}

// GetQuest returns the complete quest model for a character
func (m *ProcessorImpl) GetQuest(characterId uint32, questId uint32) model.Provider[quest.Model] {
	if m.GetQuestFunc != nil {
		return m.GetQuestFunc(characterId, questId)
	}
	return func() (quest.Model, error) {
		return quest.NewModel(questId, quest.UNDEFINED), nil
	}
}