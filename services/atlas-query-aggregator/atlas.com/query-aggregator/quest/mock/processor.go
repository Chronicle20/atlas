package mock

import (
	"atlas-query-aggregator/quest"

	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorImpl is a mock implementation of the quest.Processor interface
type ProcessorImpl struct {
	GetQuestStateFunc                 func(characterId uint32, questId uint32) model.Provider[quest.State]
	GetQuestStatusFunc                func(characterId uint32, questId uint32) model.Provider[quest.QuestStatus]
	GetQuestProgressFunc              func(characterId uint32, questId uint32, infoNumber uint32) model.Provider[int]
	GetQuestFunc                      func(characterId uint32, questId uint32) model.Provider[quest.Model]
	GetQuestsByCharacterFunc          func(characterId uint32) model.Provider[[]quest.Model]
	GetStartedQuestsByCharacterFunc   func(characterId uint32) model.Provider[[]quest.Model]
	GetCompletedQuestsByCharacterFunc func(characterId uint32) model.Provider[[]quest.Model]
}

// GetQuestState returns the state of a quest for a character
func (m *ProcessorImpl) GetQuestState(characterId uint32, questId uint32) model.Provider[quest.State] {
	if m.GetQuestStateFunc != nil {
		return m.GetQuestStateFunc(characterId, questId)
	}
	return func() (quest.State, error) {
		return quest.StateNotStarted, nil
	}
}

// GetQuestStatus returns the status of a quest for a character (alias for GetQuestState)
func (m *ProcessorImpl) GetQuestStatus(characterId uint32, questId uint32) model.Provider[quest.QuestStatus] {
	if m.GetQuestStatusFunc != nil {
		return m.GetQuestStatusFunc(characterId, questId)
	}
	// Fall back to GetQuestState if GetQuestStatusFunc is not set
	if m.GetQuestStateFunc != nil {
		return m.GetQuestStateFunc(characterId, questId)
	}
	return func() (quest.QuestStatus, error) {
		return quest.StateNotStarted, nil
	}
}

// GetQuestProgress returns the progress of a quest for a specific info number
func (m *ProcessorImpl) GetQuestProgress(characterId uint32, questId uint32, infoNumber uint32) model.Provider[int] {
	if m.GetQuestProgressFunc != nil {
		return m.GetQuestProgressFunc(characterId, questId, infoNumber)
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
		return quest.NewModel(characterId, questId, quest.StateNotStarted), nil
	}
}

// GetQuestsByCharacter returns all quests for a character
func (m *ProcessorImpl) GetQuestsByCharacter(characterId uint32) model.Provider[[]quest.Model] {
	if m.GetQuestsByCharacterFunc != nil {
		return m.GetQuestsByCharacterFunc(characterId)
	}
	return func() ([]quest.Model, error) {
		return nil, nil
	}
}

// GetStartedQuestsByCharacter returns all started quests for a character
func (m *ProcessorImpl) GetStartedQuestsByCharacter(characterId uint32) model.Provider[[]quest.Model] {
	if m.GetStartedQuestsByCharacterFunc != nil {
		return m.GetStartedQuestsByCharacterFunc(characterId)
	}
	return func() ([]quest.Model, error) {
		return nil, nil
	}
}

// GetCompletedQuestsByCharacter returns all completed quests for a character
func (m *ProcessorImpl) GetCompletedQuestsByCharacter(characterId uint32) model.Provider[[]quest.Model] {
	if m.GetCompletedQuestsByCharacterFunc != nil {
		return m.GetCompletedQuestsByCharacterFunc(characterId)
	}
	return func() ([]quest.Model, error) {
		return nil, nil
	}
}