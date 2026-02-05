package mock

import (
	"atlas-login/character"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
)

// MockProcessor is a mock implementation of character.Processor for testing
type MockProcessor struct {
	IsValidNameFunc               func(name string) (bool, error)
	ByAccountAndWorldProviderFunc func(decorators ...model.Decorator[character.Model]) func(accountId uint32, worldId world.Id) model.Provider[[]character.Model]
	GetForWorldFunc               func(decorators ...model.Decorator[character.Model]) func(accountId uint32, worldId world.Id) ([]character.Model, error)
	ByNameProviderFunc            func(decorators ...model.Decorator[character.Model]) func(name string) model.Provider[[]character.Model]
	GetByNameFunc                 func(decorators ...model.Decorator[character.Model]) func(name string) ([]character.Model, error)
	ByIdProviderFunc              func(decorators ...model.Decorator[character.Model]) func(id uint32) model.Provider[character.Model]
	GetByIdFunc                   func(decorators ...model.Decorator[character.Model]) func(id uint32) (character.Model, error)
	InventoryDecoratorFunc        func() model.Decorator[character.Model]
	DeleteByIdFunc                func(characterId uint32) error
}

// IsValidName implements character.Processor
func (m *MockProcessor) IsValidName(name string) (bool, error) {
	if m.IsValidNameFunc != nil {
		return m.IsValidNameFunc(name)
	}
	return true, nil
}

// ByAccountAndWorldProvider implements character.Processor
func (m *MockProcessor) ByAccountAndWorldProvider(decorators ...model.Decorator[character.Model]) func(accountId uint32, worldId world.Id) model.Provider[[]character.Model] {
	if m.ByAccountAndWorldProviderFunc != nil {
		return m.ByAccountAndWorldProviderFunc(decorators...)
	}
	return func(accountId uint32, worldId world.Id) model.Provider[[]character.Model] {
		return func() ([]character.Model, error) {
			return []character.Model{}, nil
		}
	}
}

// GetForWorld implements character.Processor
func (m *MockProcessor) GetForWorld(decorators ...model.Decorator[character.Model]) func(accountId uint32, worldId world.Id) ([]character.Model, error) {
	if m.GetForWorldFunc != nil {
		return m.GetForWorldFunc(decorators...)
	}
	return func(accountId uint32, worldId world.Id) ([]character.Model, error) {
		return []character.Model{}, nil
	}
}

// ByNameProvider implements character.Processor
func (m *MockProcessor) ByNameProvider(decorators ...model.Decorator[character.Model]) func(name string) model.Provider[[]character.Model] {
	if m.ByNameProviderFunc != nil {
		return m.ByNameProviderFunc(decorators...)
	}
	return func(name string) model.Provider[[]character.Model] {
		return func() ([]character.Model, error) {
			return []character.Model{}, nil
		}
	}
}

// GetByName implements character.Processor
func (m *MockProcessor) GetByName(decorators ...model.Decorator[character.Model]) func(name string) ([]character.Model, error) {
	if m.GetByNameFunc != nil {
		return m.GetByNameFunc(decorators...)
	}
	return func(name string) ([]character.Model, error) {
		return []character.Model{}, nil
	}
}

// ByIdProvider implements character.Processor
func (m *MockProcessor) ByIdProvider(decorators ...model.Decorator[character.Model]) func(id uint32) model.Provider[character.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(decorators...)
	}
	return func(id uint32) model.Provider[character.Model] {
		return func() (character.Model, error) {
			return character.Model{}, nil
		}
	}
}

// GetById implements character.Processor
func (m *MockProcessor) GetById(decorators ...model.Decorator[character.Model]) func(id uint32) (character.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(decorators...)
	}
	return func(id uint32) (character.Model, error) {
		return character.Model{}, nil
	}
}

// InventoryDecorator implements character.Processor
func (m *MockProcessor) InventoryDecorator() model.Decorator[character.Model] {
	if m.InventoryDecoratorFunc != nil {
		return m.InventoryDecoratorFunc()
	}
	return func(c character.Model) character.Model {
		return c
	}
}

// DeleteById implements character.Processor
func (m *MockProcessor) DeleteById(characterId uint32) error {
	if m.DeleteByIdFunc != nil {
		return m.DeleteByIdFunc(characterId)
	}
	return nil
}

// Verify MockProcessor implements character.Processor
var _ character.Processor = (*MockProcessor)(nil)
