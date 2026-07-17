package mock

import (
	"atlas-configurations/services"
	"atlas-configurations/services/service"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByIdProviderFunc func(id uuid.UUID) model.Provider[interface{}]
	AllProviderFunc  func() model.Provider[[]interface{}]
	GetAllFunc       func() ([]interface{}, error)
	GetByIdFunc      func(id uuid.UUID) (interface{}, error)
	CreateFunc       func(input service.InputRestModel) (uuid.UUID, error)
	UpdateByIdFunc   func(serviceId uuid.UUID, input service.InputRestModel) error
	DeleteByIdFunc   func(serviceId uuid.UUID) error
}

var _ services.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByIdProvider(id uuid.UUID) model.Provider[interface{}] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return model.FixedProvider[interface{}](nil)
}

func (m *ProcessorMock) AllProvider() model.Provider[[]interface{}] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc()
	}
	return model.FixedProvider([]interface{}{})
}

func (m *ProcessorMock) GetAll() ([]interface{}, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return nil, nil
}

func (m *ProcessorMock) GetById(id uuid.UUID) (interface{}, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return nil, nil
}

func (m *ProcessorMock) Create(input service.InputRestModel) (uuid.UUID, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(input)
	}
	return uuid.Nil, nil
}

func (m *ProcessorMock) UpdateById(serviceId uuid.UUID, input service.InputRestModel) error {
	if m.UpdateByIdFunc != nil {
		return m.UpdateByIdFunc(serviceId, input)
	}
	return nil
}

func (m *ProcessorMock) DeleteById(serviceId uuid.UUID) error {
	if m.DeleteByIdFunc != nil {
		return m.DeleteByIdFunc(serviceId)
	}
	return nil
}
