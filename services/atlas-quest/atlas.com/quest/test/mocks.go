package test

import (
	dataquest "atlas-quest/data/quest"
	"atlas-quest/data/validation"
	"errors"
)

// MockDataProcessor implements dataquest.Processor for testing
type MockDataProcessor struct {
	Definitions    map[uint32]dataquest.RestModel
	AutoStartQuests []dataquest.RestModel
	GetError       error
}

// NewMockDataProcessor creates a new mock data processor
func NewMockDataProcessor() *MockDataProcessor {
	return &MockDataProcessor{
		Definitions:    make(map[uint32]dataquest.RestModel),
		AutoStartQuests: make([]dataquest.RestModel, 0),
	}
}

// AddQuestDefinition adds a quest definition to the mock
func (m *MockDataProcessor) AddQuestDefinition(questId uint32, def dataquest.RestModel) {
	def.Id = questId
	m.Definitions[questId] = def
}

// GetQuestDefinition implements dataquest.Processor
func (m *MockDataProcessor) GetQuestDefinition(questId uint32) (dataquest.RestModel, error) {
	if m.GetError != nil {
		return dataquest.RestModel{}, m.GetError
	}
	if def, ok := m.Definitions[questId]; ok {
		return def, nil
	}
	return dataquest.RestModel{}, errors.New("quest not found")
}

// GetAutoStartQuests implements dataquest.Processor
func (m *MockDataProcessor) GetAutoStartQuests(mapId uint32) ([]dataquest.RestModel, error) {
	if m.GetError != nil {
		return nil, m.GetError
	}
	if mapId == 0 {
		return m.AutoStartQuests, nil
	}
	var filtered []dataquest.RestModel
	for _, q := range m.AutoStartQuests {
		if q.StartRequirements.NormalAutoStart || q.Area == mapId {
			filtered = append(filtered, q)
		}
	}
	return filtered, nil
}

// MockValidationProcessor implements validation.Processor for testing
type MockValidationProcessor struct {
	StartValidationResult bool
	StartFailedConditions []string
	StartError            error
	EndValidationResult   bool
	EndFailedConditions   []string
	EndError              error
}

// NewMockValidationProcessor creates a new mock validation processor that passes all validations
func NewMockValidationProcessor() *MockValidationProcessor {
	return &MockValidationProcessor{
		StartValidationResult: true,
		StartFailedConditions: nil,
		EndValidationResult:   true,
		EndFailedConditions:   nil,
	}
}

// ValidateStartRequirements implements validation.Processor
func (m *MockValidationProcessor) ValidateStartRequirements(characterId uint32, questDef dataquest.RestModel) (bool, []string, error) {
	if m.StartError != nil {
		return false, nil, m.StartError
	}
	return m.StartValidationResult, m.StartFailedConditions, nil
}

// ValidateEndRequirements implements validation.Processor
func (m *MockValidationProcessor) ValidateEndRequirements(characterId uint32, questDef dataquest.RestModel) (bool, []string, error) {
	if m.EndError != nil {
		return false, nil, m.EndError
	}
	return m.EndValidationResult, m.EndFailedConditions, nil
}

// Ensure mocks implement interfaces
var _ dataquest.Processor = (*MockDataProcessor)(nil)
var _ validation.Processor = (*MockValidationProcessor)(nil)

// CreateSimpleQuestDefinition creates a basic quest definition for testing
func CreateSimpleQuestDefinition(questId uint32) dataquest.RestModel {
	return dataquest.RestModel{
		Id:           questId,
		Name:         "Test Quest",
		AutoComplete: false,
		StartRequirements: dataquest.RequirementsRestModel{},
		EndRequirements:   dataquest.RequirementsRestModel{},
		StartActions:      dataquest.ActionsRestModel{},
		EndActions:        dataquest.ActionsRestModel{},
	}
}

// CreateQuestWithMobRequirement creates a quest with mob kill requirements
func CreateQuestWithMobRequirement(questId uint32, mobId uint32, count uint32) dataquest.RestModel {
	def := CreateSimpleQuestDefinition(questId)
	def.EndRequirements.Mobs = []dataquest.MobRequirement{
		{Id: mobId, Count: count},
	}
	return def
}

// CreateQuestWithMapRequirement creates a quest with map visit requirements
func CreateQuestWithMapRequirement(questId uint32, mapIds []uint32) dataquest.RestModel {
	def := CreateSimpleQuestDefinition(questId)
	def.EndRequirements.FieldEnter = mapIds
	return def
}

// CreateQuestWithChain creates a quest that chains to another quest
func CreateQuestWithChain(questId uint32, nextQuestId uint32) dataquest.RestModel {
	def := CreateSimpleQuestDefinition(questId)
	def.EndActions.NextQuest = nextQuestId
	return def
}

// CreateRepeatableQuest creates a quest that can be repeated after an interval
func CreateRepeatableQuest(questId uint32, intervalMinutes uint32) dataquest.RestModel {
	def := CreateSimpleQuestDefinition(questId)
	def.StartRequirements.Interval = intervalMinutes
	return def
}

// CreateAutoCompleteQuest creates a quest that auto-completes
func CreateAutoCompleteQuest(questId uint32) dataquest.RestModel {
	def := CreateSimpleQuestDefinition(questId)
	def.AutoComplete = true
	return def
}

// CreateTimeLimitedQuest creates a quest with a time limit
func CreateTimeLimitedQuest(questId uint32, timeLimitSeconds uint32) dataquest.RestModel {
	def := CreateSimpleQuestDefinition(questId)
	def.TimeLimit = timeLimitSeconds
	return def
}

// CreateQuestWithItemRequirement creates a quest with item collection requirements
func CreateQuestWithItemRequirement(questId uint32, itemId uint32, count int32) dataquest.RestModel {
	def := CreateSimpleQuestDefinition(questId)
	def.EndRequirements.Items = []dataquest.ItemRequirement{
		{Id: itemId, Count: count},
	}
	return def
}
