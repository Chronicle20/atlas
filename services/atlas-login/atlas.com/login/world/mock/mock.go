package mock

import (
	"atlas-login/world"

	world2 "github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
)

// MockProcessor is a mock implementation of world.Processor for testing
type MockProcessor struct {
	GetAllFunc            func() ([]world.Model, error)
	AllProviderFunc       func() model.Provider[[]world.Model]
	GetByIdFunc           func(worldId world2.Id) (world.Model, error)
	ByIdModelProviderFunc func(worldId world2.Id) model.Provider[world.Model]
	GetCapacityStatusFunc func(worldId world2.Id) world.Status
}

// GetAll implements world.Processor
func (m *MockProcessor) GetAll() ([]world.Model, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return []world.Model{}, nil
}

// AllProvider implements world.Processor
func (m *MockProcessor) AllProvider() model.Provider[[]world.Model] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc()
	}
	return func() ([]world.Model, error) {
		return []world.Model{}, nil
	}
}

// GetById implements world.Processor
func (m *MockProcessor) GetById(worldId world2.Id) (world.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(worldId)
	}
	return world.Model{}, nil
}

// ByIdModelProvider implements world.Processor
func (m *MockProcessor) ByIdModelProvider(worldId world2.Id) model.Provider[world.Model] {
	if m.ByIdModelProviderFunc != nil {
		return m.ByIdModelProviderFunc(worldId)
	}
	return func() (world.Model, error) {
		return world.Model{}, nil
	}
}

// GetCapacityStatus implements world.Processor
func (m *MockProcessor) GetCapacityStatus(worldId world2.Id) world.Status {
	if m.GetCapacityStatusFunc != nil {
		return m.GetCapacityStatusFunc(worldId)
	}
	return world.StatusNormal
}

// Verify MockProcessor implements world.Processor
var _ world.Processor = (*MockProcessor)(nil)
