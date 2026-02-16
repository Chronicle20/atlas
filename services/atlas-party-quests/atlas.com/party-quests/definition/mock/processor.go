package mock

import (
	"atlas-party-quests/definition"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	CreateFunc              func(m definition.Model) (definition.Model, error)
	UpdateFunc              func(id uuid.UUID, m definition.Model) (definition.Model, error)
	DeleteFunc              func(id uuid.UUID) error
	ByIdProviderFunc        func(id uuid.UUID) model.Provider[definition.Model]
	ByQuestIdProviderFunc   func(questId string) model.Provider[definition.Model]
	AllProviderFunc         func() model.Provider[[]definition.Model]
	DeleteAllForTenantFunc  func() (int64, error)
	SeedFunc                func() (definition.SeedResult, error)
	ValidateDefinitionsFunc func() []definition.ValidationResult
}

func (m *ProcessorMock) Create(dm definition.Model) (definition.Model, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(dm)
	}
	return definition.Model{}, nil
}

func (m *ProcessorMock) Update(id uuid.UUID, dm definition.Model) (definition.Model, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(id, dm)
	}
	return definition.Model{}, nil
}

func (m *ProcessorMock) Delete(id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *ProcessorMock) ByIdProvider(id uuid.UUID) model.Provider[definition.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return func() (definition.Model, error) {
		return definition.Model{}, nil
	}
}

func (m *ProcessorMock) ByQuestIdProvider(questId string) model.Provider[definition.Model] {
	if m.ByQuestIdProviderFunc != nil {
		return m.ByQuestIdProviderFunc(questId)
	}
	return func() (definition.Model, error) {
		return definition.Model{}, nil
	}
}

func (m *ProcessorMock) AllProvider() model.Provider[[]definition.Model] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc()
	}
	return func() ([]definition.Model, error) {
		return []definition.Model{}, nil
	}
}

func (m *ProcessorMock) DeleteAllForTenant() (int64, error) {
	if m.DeleteAllForTenantFunc != nil {
		return m.DeleteAllForTenantFunc()
	}
	return 0, nil
}

func (m *ProcessorMock) Seed() (definition.SeedResult, error) {
	if m.SeedFunc != nil {
		return m.SeedFunc()
	}
	return definition.SeedResult{}, nil
}

func (m *ProcessorMock) ValidateDefinitions() []definition.ValidationResult {
	if m.ValidateDefinitionsFunc != nil {
		return m.ValidateDefinitionsFunc()
	}
	return nil
}
