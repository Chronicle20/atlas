package mock

import (
	"atlas-npc-conversations/conversation/npc"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

// ProcessorMock is a mock implementation of the npc.Processor interface
type ProcessorMock struct {
	// CreateFunc is a function field for the Create method
	CreateFunc func(m npc.Model) (npc.Model, error)

	// UpdateFunc is a function field for the Update method
	UpdateFunc func(id uuid.UUID, m npc.Model) (npc.Model, error)

	// DeleteFunc is a function field for the Delete method
	DeleteFunc func(id uuid.UUID) error

	// ByIdProviderFunc is a function field for the ByIdProvider method
	ByIdProviderFunc func(id uuid.UUID) model.Provider[npc.Model]

	// ByNpcIdProviderFunc is a function field for the ByNpcIdProvider method
	ByNpcIdProviderFunc func(npcId uint32) model.Provider[npc.Model]

	// AllByNpcIdProviderFunc is a function field for the AllByNpcIdProvider method
	AllByNpcIdProviderFunc func(npcId uint32) model.Provider[[]npc.Model]

	// AllProviderFunc is a function field for the AllProvider method
	AllProviderFunc func() model.Provider[[]npc.Model]

	// DeleteAllForTenantFunc is a function field for the DeleteAllForTenant method
	DeleteAllForTenantFunc func() (int64, error)

	// SeedFunc is a function field for the Seed method
	SeedFunc func() (npc.SeedResult, error)
}

// Create is a mock implementation of the npc.Processor.Create method
func (m *ProcessorMock) Create(model npc.Model) (npc.Model, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(model)
	}
	return npc.Model{}, nil
}

// Update is a mock implementation of the npc.Processor.Update method
func (m *ProcessorMock) Update(id uuid.UUID, model npc.Model) (npc.Model, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(id, model)
	}
	return npc.Model{}, nil
}

// Delete is a mock implementation of the npc.Processor.Delete method
func (m *ProcessorMock) Delete(id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

// ByIdProvider is a mock implementation of the npc.Processor.ByIdProvider method
func (m *ProcessorMock) ByIdProvider(id uuid.UUID) model.Provider[npc.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return func() (npc.Model, error) {
		return npc.Model{}, nil
	}
}

// ByNpcIdProvider is a mock implementation of the npc.Processor.ByNpcIdProvider method
func (m *ProcessorMock) ByNpcIdProvider(npcId uint32) model.Provider[npc.Model] {
	if m.ByNpcIdProviderFunc != nil {
		return m.ByNpcIdProviderFunc(npcId)
	}
	return func() (npc.Model, error) {
		return npc.Model{}, nil
	}
}

// AllByNpcIdProvider is a mock implementation of the npc.Processor.AllByNpcIdProvider method
func (m *ProcessorMock) AllByNpcIdProvider(npcId uint32) model.Provider[[]npc.Model] {
	if m.AllByNpcIdProviderFunc != nil {
		return m.AllByNpcIdProviderFunc(npcId)
	}
	return func() ([]npc.Model, error) {
		return []npc.Model{}, nil
	}
}

// AllProvider is a mock implementation of the npc.Processor.AllProvider method
func (m *ProcessorMock) AllProvider() model.Provider[[]npc.Model] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc()
	}
	return func() ([]npc.Model, error) {
		return []npc.Model{}, nil
	}
}

// DeleteAllForTenant is a mock implementation of the npc.Processor.DeleteAllForTenant method
func (m *ProcessorMock) DeleteAllForTenant() (int64, error) {
	if m.DeleteAllForTenantFunc != nil {
		return m.DeleteAllForTenantFunc()
	}
	return 0, nil
}

// Seed is a mock implementation of the npc.Processor.Seed method
func (m *ProcessorMock) Seed() (npc.SeedResult, error) {
	if m.SeedFunc != nil {
		return m.SeedFunc()
	}
	return npc.SeedResult{}, nil
}
