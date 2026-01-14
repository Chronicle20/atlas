package mock

import (
	"atlas-tenants/kafka/message"
	"atlas-tenants/tenant"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

// Compile-time interface compliance check
var _ tenant.Processor = (*ProcessorMock)(nil)

// ProcessorMock is a mock implementation of the tenant.Processor interface
type ProcessorMock struct {
	CreateFunc       func(mb *message.Buffer) func(name string, region string, majorVersion uint16, minorVersion uint16) (tenant.Model, error)
	CreateAndEmitFunc func(name string, region string, majorVersion uint16, minorVersion uint16) (tenant.Model, error)
	UpdateFunc       func(mb *message.Buffer) func(id uuid.UUID, name string, region string, majorVersion uint16, minorVersion uint16) (tenant.Model, error)
	UpdateAndEmitFunc func(id uuid.UUID, name string, region string, majorVersion uint16, minorVersion uint16) (tenant.Model, error)
	DeleteFunc       func(mb *message.Buffer) func(id uuid.UUID) error
	DeleteAndEmitFunc func(id uuid.UUID) error
	GetByIdFunc      func(id uuid.UUID) (tenant.Model, error)
	GetAllFunc       func() ([]tenant.Model, error)
	ByIdProviderFunc func(id uuid.UUID) model.Provider[tenant.Model]
	AllProviderFunc  func() model.Provider[[]tenant.Model]
}

// Create is a mock implementation
func (m *ProcessorMock) Create(mb *message.Buffer) func(name string, region string, majorVersion uint16, minorVersion uint16) (tenant.Model, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(mb)
	}
	return func(name string, region string, majorVersion uint16, minorVersion uint16) (tenant.Model, error) {
		return tenant.Model{}, nil
	}
}

// CreateAndEmit is a mock implementation
func (m *ProcessorMock) CreateAndEmit(name string, region string, majorVersion uint16, minorVersion uint16) (tenant.Model, error) {
	if m.CreateAndEmitFunc != nil {
		return m.CreateAndEmitFunc(name, region, majorVersion, minorVersion)
	}
	return tenant.Model{}, nil
}

// Update is a mock implementation
func (m *ProcessorMock) Update(mb *message.Buffer) func(id uuid.UUID, name string, region string, majorVersion uint16, minorVersion uint16) (tenant.Model, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(mb)
	}
	return func(id uuid.UUID, name string, region string, majorVersion uint16, minorVersion uint16) (tenant.Model, error) {
		return tenant.Model{}, nil
	}
}

// UpdateAndEmit is a mock implementation
func (m *ProcessorMock) UpdateAndEmit(id uuid.UUID, name string, region string, majorVersion uint16, minorVersion uint16) (tenant.Model, error) {
	if m.UpdateAndEmitFunc != nil {
		return m.UpdateAndEmitFunc(id, name, region, majorVersion, minorVersion)
	}
	return tenant.Model{}, nil
}

// Delete is a mock implementation
func (m *ProcessorMock) Delete(mb *message.Buffer) func(id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(mb)
	}
	return func(id uuid.UUID) error {
		return nil
	}
}

// DeleteAndEmit is a mock implementation
func (m *ProcessorMock) DeleteAndEmit(id uuid.UUID) error {
	if m.DeleteAndEmitFunc != nil {
		return m.DeleteAndEmitFunc(id)
	}
	return nil
}

// GetById is a mock implementation
func (m *ProcessorMock) GetById(id uuid.UUID) (tenant.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return tenant.Model{}, nil
}

// GetAll is a mock implementation
func (m *ProcessorMock) GetAll() ([]tenant.Model, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return []tenant.Model{}, nil
}

// ByIdProvider is a mock implementation
func (m *ProcessorMock) ByIdProvider(id uuid.UUID) model.Provider[tenant.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return func() (tenant.Model, error) {
		return tenant.Model{}, nil
	}
}

// AllProvider is a mock implementation
func (m *ProcessorMock) AllProvider() model.Provider[[]tenant.Model] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc()
	}
	return func() ([]tenant.Model, error) {
		return []tenant.Model{}, nil
	}
}
