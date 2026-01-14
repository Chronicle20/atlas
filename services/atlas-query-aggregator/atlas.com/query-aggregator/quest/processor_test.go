package quest_test

import (
	"atlas-query-aggregator/quest"
	"atlas-query-aggregator/quest/mock"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas-model/model"
)

func TestProcessorMock_GetQuestState_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetQuestStateFunc: func(characterId uint32, questId uint32) model.Provider[quest.State] {
			return func() (quest.State, error) {
				if questId == 1001 {
					return quest.StateStarted, nil
				}
				return quest.StateNotStarted, nil
			}
		},
	}

	state, err := mockProcessor.GetQuestState(123, 1001)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if state != quest.StateStarted {
		t.Errorf("Expected state=StateStarted, got %v", state)
	}
}

func TestProcessorMock_GetQuestState_NotStarted(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetQuestStateFunc: func(characterId uint32, questId uint32) model.Provider[quest.State] {
			return func() (quest.State, error) {
				return quest.StateNotStarted, nil
			}
		},
	}

	state, err := mockProcessor.GetQuestState(123, 999)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if state != quest.StateNotStarted {
		t.Errorf("Expected state=StateNotStarted, got %v", state)
	}
}

func TestProcessorMock_GetQuestState_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetQuestStateFunc: func(characterId uint32, questId uint32) model.Provider[quest.State] {
			return func() (quest.State, error) {
				return quest.StateNotStarted, errors.New("quest service unavailable")
			}
		},
	}

	_, err := mockProcessor.GetQuestState(123, 1001)()
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestProcessorMock_GetQuestProgress_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetQuestProgressFunc: func(characterId uint32, questId uint32, infoNumber uint32) model.Provider[int] {
			return func() (int, error) {
				if questId == 1001 && infoNumber == 0 {
					return 50, nil
				}
				return 0, nil
			}
		},
	}

	progress, err := mockProcessor.GetQuestProgress(123, 1001, 0)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if progress != 50 {
		t.Errorf("Expected progress=50, got %d", progress)
	}
}

func TestProcessorMock_GetQuest_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetQuestFunc: func(characterId uint32, questId uint32) model.Provider[quest.Model] {
			return func() (quest.Model, error) {
				return quest.NewModel(characterId, questId, quest.StateStarted), nil
			}
		},
	}

	q, err := mockProcessor.GetQuest(123, 1001)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if q.CharacterId() != 123 {
		t.Errorf("Expected CharacterId=123, got %d", q.CharacterId())
	}

	if q.QuestId() != 1001 {
		t.Errorf("Expected QuestId=1001, got %d", q.QuestId())
	}

	if q.State() != quest.StateStarted {
		t.Errorf("Expected State=StateStarted, got %v", q.State())
	}
}

func TestProcessorMock_GetQuestsByCharacter_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetQuestsByCharacterFunc: func(characterId uint32) model.Provider[[]quest.Model] {
			return func() ([]quest.Model, error) {
				return []quest.Model{
					quest.NewModel(characterId, 1001, quest.StateStarted),
					quest.NewModel(characterId, 1002, quest.StateCompleted),
				}, nil
			}
		},
	}

	quests, err := mockProcessor.GetQuestsByCharacter(123)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(quests) != 2 {
		t.Errorf("Expected 2 quests, got %d", len(quests))
	}
}

func TestProcessorMock_GetStartedQuestsByCharacter_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetStartedQuestsByCharacterFunc: func(characterId uint32) model.Provider[[]quest.Model] {
			return func() ([]quest.Model, error) {
				return []quest.Model{
					quest.NewModel(characterId, 1001, quest.StateStarted),
				}, nil
			}
		},
	}

	quests, err := mockProcessor.GetStartedQuestsByCharacter(123)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(quests) != 1 {
		t.Errorf("Expected 1 quest, got %d", len(quests))
	}

	if quests[0].State() != quest.StateStarted {
		t.Errorf("Expected quest state=StateStarted, got %v", quests[0].State())
	}
}

func TestProcessorMock_GetCompletedQuestsByCharacter_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetCompletedQuestsByCharacterFunc: func(characterId uint32) model.Provider[[]quest.Model] {
			return func() ([]quest.Model, error) {
				return []quest.Model{
					quest.NewModel(characterId, 1002, quest.StateCompleted),
				}, nil
			}
		},
	}

	quests, err := mockProcessor.GetCompletedQuestsByCharacter(123)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(quests) != 1 {
		t.Errorf("Expected 1 quest, got %d", len(quests))
	}

	if quests[0].State() != quest.StateCompleted {
		t.Errorf("Expected quest state=StateCompleted, got %v", quests[0].State())
	}
}

func TestProcessorMock_DefaultBehavior(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{}

	// Test default GetQuestState returns StateNotStarted
	state, err := mockProcessor.GetQuestState(123, 1001)()
	if err != nil {
		t.Errorf("Expected no error from default GetQuestState, got %v", err)
	}

	if state != quest.StateNotStarted {
		t.Errorf("Expected default state=StateNotStarted, got %v", state)
	}

	// Test default GetQuestProgress returns 0
	progress, err := mockProcessor.GetQuestProgress(123, 1001, 0)()
	if err != nil {
		t.Errorf("Expected no error from default GetQuestProgress, got %v", err)
	}

	if progress != 0 {
		t.Errorf("Expected default progress=0, got %d", progress)
	}

	// Test default GetQuest returns empty model
	q, err := mockProcessor.GetQuest(123, 1001)()
	if err != nil {
		t.Errorf("Expected no error from default GetQuest, got %v", err)
	}

	if q.State() != quest.StateNotStarted {
		t.Errorf("Expected default quest state=StateNotStarted, got %v", q.State())
	}

	// Test default GetQuestsByCharacter returns nil
	quests, err := mockProcessor.GetQuestsByCharacter(123)()
	if err != nil {
		t.Errorf("Expected no error from default GetQuestsByCharacter, got %v", err)
	}

	if quests != nil {
		t.Errorf("Expected default quests=nil, got %v", quests)
	}
}
