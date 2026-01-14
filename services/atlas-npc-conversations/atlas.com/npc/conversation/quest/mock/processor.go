package mock

import (
	"atlas-npc-conversations/conversation/quest"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

// ProcessorMock is a mock implementation of the quest.Processor interface
type ProcessorMock struct {
	// CreateFunc is a function field for the Create method
	CreateFunc func(m quest.Model) (quest.Model, error)

	// UpdateFunc is a function field for the Update method
	UpdateFunc func(id uuid.UUID, m quest.Model) (quest.Model, error)

	// DeleteFunc is a function field for the Delete method
	DeleteFunc func(id uuid.UUID) error

	// ByIdProviderFunc is a function field for the ByIdProvider method
	ByIdProviderFunc func(id uuid.UUID) model.Provider[quest.Model]

	// ByQuestIdProviderFunc is a function field for the ByQuestIdProvider method
	ByQuestIdProviderFunc func(questId uint32) model.Provider[quest.Model]

	// AllProviderFunc is a function field for the AllProvider method
	AllProviderFunc func() model.Provider[[]quest.Model]

	// DeleteAllForTenantFunc is a function field for the DeleteAllForTenant method
	DeleteAllForTenantFunc func() (int64, error)

	// SeedFunc is a function field for the Seed method
	SeedFunc func() (quest.SeedResult, error)

	// GetStateMachineForCharacterFunc is a function field for the GetStateMachineForCharacter method
	GetStateMachineForCharacterFunc func(questId uint32, characterId uint32) (quest.StateMachine, error)
}

// Create is a mock implementation of the quest.Processor.Create method
func (m *ProcessorMock) Create(model quest.Model) (quest.Model, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(model)
	}
	return quest.Model{}, nil
}

// Update is a mock implementation of the quest.Processor.Update method
func (m *ProcessorMock) Update(id uuid.UUID, model quest.Model) (quest.Model, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(id, model)
	}
	return quest.Model{}, nil
}

// Delete is a mock implementation of the quest.Processor.Delete method
func (m *ProcessorMock) Delete(id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

// ByIdProvider is a mock implementation of the quest.Processor.ByIdProvider method
func (m *ProcessorMock) ByIdProvider(id uuid.UUID) model.Provider[quest.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return func() (quest.Model, error) {
		return quest.Model{}, nil
	}
}

// ByQuestIdProvider is a mock implementation of the quest.Processor.ByQuestIdProvider method
func (m *ProcessorMock) ByQuestIdProvider(questId uint32) model.Provider[quest.Model] {
	if m.ByQuestIdProviderFunc != nil {
		return m.ByQuestIdProviderFunc(questId)
	}
	return func() (quest.Model, error) {
		return quest.Model{}, nil
	}
}

// AllProvider is a mock implementation of the quest.Processor.AllProvider method
func (m *ProcessorMock) AllProvider() model.Provider[[]quest.Model] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc()
	}
	return func() ([]quest.Model, error) {
		return []quest.Model{}, nil
	}
}

// DeleteAllForTenant is a mock implementation of the quest.Processor.DeleteAllForTenant method
func (m *ProcessorMock) DeleteAllForTenant() (int64, error) {
	if m.DeleteAllForTenantFunc != nil {
		return m.DeleteAllForTenantFunc()
	}
	return 0, nil
}

// Seed is a mock implementation of the quest.Processor.Seed method
func (m *ProcessorMock) Seed() (quest.SeedResult, error) {
	if m.SeedFunc != nil {
		return m.SeedFunc()
	}
	return quest.SeedResult{}, nil
}

// GetStateMachineForCharacter is a mock implementation of the quest.Processor.GetStateMachineForCharacter method
func (m *ProcessorMock) GetStateMachineForCharacter(questId uint32, characterId uint32) (quest.StateMachine, error) {
	if m.GetStateMachineForCharacterFunc != nil {
		return m.GetStateMachineForCharacterFunc(questId, characterId)
	}
	return quest.StateMachine{}, nil
}
