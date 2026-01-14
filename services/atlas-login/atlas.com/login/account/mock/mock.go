package mock

import (
	"atlas-login/account"

	"github.com/Chronicle20/atlas-model/model"
)

// MockProcessor is a mock implementation of account.Processor for testing
type MockProcessor struct {
	ForAccountByNameFunc   func(name string, operator model.Operator[account.Model])
	ForAccountByIdFunc     func(id uint32, operator model.Operator[account.Model])
	ByNameModelProviderFunc func(name string) model.Provider[account.Model]
	ByIdModelProviderFunc   func(id uint32) model.Provider[account.Model]
	AllProviderFunc         func() model.Provider[[]account.Model]
	GetByIdFunc             func(id uint32) (account.Model, error)
	GetByNameFunc           func(name string) (account.Model, error)
	IsLoggedInFunc          func(id uint32) bool
	InitializeRegistryFunc  func() error
	UpdatePinFunc           func(id uint32, pin string) error
	UpdatePicFunc           func(id uint32, pic string) error
	UpdateTosFunc           func(id uint32, tos bool) error
	UpdateGenderFunc        func(id uint32, gender byte) error
}

// ForAccountByName implements account.Processor
func (m *MockProcessor) ForAccountByName(name string, operator model.Operator[account.Model]) {
	if m.ForAccountByNameFunc != nil {
		m.ForAccountByNameFunc(name, operator)
	}
}

// ForAccountById implements account.Processor
func (m *MockProcessor) ForAccountById(id uint32, operator model.Operator[account.Model]) {
	if m.ForAccountByIdFunc != nil {
		m.ForAccountByIdFunc(id, operator)
	}
}

// ByNameModelProvider implements account.Processor
func (m *MockProcessor) ByNameModelProvider(name string) model.Provider[account.Model] {
	if m.ByNameModelProviderFunc != nil {
		return m.ByNameModelProviderFunc(name)
	}
	return func() (account.Model, error) {
		return account.Model{}, nil
	}
}

// ByIdModelProvider implements account.Processor
func (m *MockProcessor) ByIdModelProvider(id uint32) model.Provider[account.Model] {
	if m.ByIdModelProviderFunc != nil {
		return m.ByIdModelProviderFunc(id)
	}
	return func() (account.Model, error) {
		return account.Model{}, nil
	}
}

// AllProvider implements account.Processor
func (m *MockProcessor) AllProvider() model.Provider[[]account.Model] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc()
	}
	return func() ([]account.Model, error) {
		return []account.Model{}, nil
	}
}

// GetById implements account.Processor
func (m *MockProcessor) GetById(id uint32) (account.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return account.Model{}, nil
}

// GetByName implements account.Processor
func (m *MockProcessor) GetByName(name string) (account.Model, error) {
	if m.GetByNameFunc != nil {
		return m.GetByNameFunc(name)
	}
	return account.Model{}, nil
}

// IsLoggedIn implements account.Processor
func (m *MockProcessor) IsLoggedIn(id uint32) bool {
	if m.IsLoggedInFunc != nil {
		return m.IsLoggedInFunc(id)
	}
	return false
}

// InitializeRegistry implements account.Processor
func (m *MockProcessor) InitializeRegistry() error {
	if m.InitializeRegistryFunc != nil {
		return m.InitializeRegistryFunc()
	}
	return nil
}

// UpdatePin implements account.Processor
func (m *MockProcessor) UpdatePin(id uint32, pin string) error {
	if m.UpdatePinFunc != nil {
		return m.UpdatePinFunc(id, pin)
	}
	return nil
}

// UpdatePic implements account.Processor
func (m *MockProcessor) UpdatePic(id uint32, pic string) error {
	if m.UpdatePicFunc != nil {
		return m.UpdatePicFunc(id, pic)
	}
	return nil
}

// UpdateTos implements account.Processor
func (m *MockProcessor) UpdateTos(id uint32, tos bool) error {
	if m.UpdateTosFunc != nil {
		return m.UpdateTosFunc(id, tos)
	}
	return nil
}

// UpdateGender implements account.Processor
func (m *MockProcessor) UpdateGender(id uint32, gender byte) error {
	if m.UpdateGenderFunc != nil {
		return m.UpdateGenderFunc(id, gender)
	}
	return nil
}

// Verify MockProcessor implements account.Processor
var _ account.Processor = (*MockProcessor)(nil)
