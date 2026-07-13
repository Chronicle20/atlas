package mock

import (
	"atlas-configurations/tenants"
	"atlas-configurations/tenants/characters/preset"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	WithValidatorFunc              func(v *preset.Validator) tenants.Processor
	ByIdProviderFunc               func(id uuid.UUID) model.Provider[tenants.RestModel]
	ByRegionAndVersionProviderFunc func(region string, majorVersion uint16, minorVersion uint16) model.Provider[tenants.RestModel]
	AllProviderFunc                func() model.Provider[[]tenants.RestModel]
	GetAllFunc                     func() ([]tenants.RestModel, error)
	GetByIdFunc                    func(id uuid.UUID) (tenants.RestModel, error)
	GetByRegionAndVersionFunc      func(region string, majorVersion uint16, minorVersion uint16) (tenants.RestModel, error)
	UpdateByIdFunc                 func(tenantId uuid.UUID, input tenants.RestModel) error
	DeleteByIdFunc                 func(tenantId uuid.UUID) error
	CreateFunc                     func(input tenants.RestModel) (uuid.UUID, error)
}

var _ tenants.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) WithValidator(v *preset.Validator) tenants.Processor {
	if m.WithValidatorFunc != nil {
		return m.WithValidatorFunc(v)
	}
	return m
}

func (m *ProcessorMock) ByIdProvider(id uuid.UUID) model.Provider[tenants.RestModel] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return model.FixedProvider(tenants.RestModel{})
}

func (m *ProcessorMock) ByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[tenants.RestModel] {
	if m.ByRegionAndVersionProviderFunc != nil {
		return m.ByRegionAndVersionProviderFunc(region, majorVersion, minorVersion)
	}
	return model.FixedProvider(tenants.RestModel{})
}

func (m *ProcessorMock) AllProvider() model.Provider[[]tenants.RestModel] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc()
	}
	return model.FixedProvider([]tenants.RestModel{})
}

func (m *ProcessorMock) GetAll() ([]tenants.RestModel, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return nil, nil
}

func (m *ProcessorMock) GetById(id uuid.UUID) (tenants.RestModel, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return tenants.RestModel{}, nil
}

func (m *ProcessorMock) GetByRegionAndVersion(region string, majorVersion uint16, minorVersion uint16) (tenants.RestModel, error) {
	if m.GetByRegionAndVersionFunc != nil {
		return m.GetByRegionAndVersionFunc(region, majorVersion, minorVersion)
	}
	return tenants.RestModel{}, nil
}

func (m *ProcessorMock) UpdateById(tenantId uuid.UUID, input tenants.RestModel) error {
	if m.UpdateByIdFunc != nil {
		return m.UpdateByIdFunc(tenantId, input)
	}
	return nil
}

func (m *ProcessorMock) DeleteById(tenantId uuid.UUID) error {
	if m.DeleteByIdFunc != nil {
		return m.DeleteByIdFunc(tenantId)
	}
	return nil
}

func (m *ProcessorMock) Create(input tenants.RestModel) (uuid.UUID, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(input)
	}
	return uuid.Nil, nil
}
