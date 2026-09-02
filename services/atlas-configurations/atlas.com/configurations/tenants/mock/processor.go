package mock

import (
	"atlas-configurations/templates"
	"atlas-configurations/tenants"
	"atlas-configurations/tenants/characters/preset"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	WithValidatorFunc              func(v *preset.Validator) tenants.Processor
	WithTemplatesFunc              func(tp templates.Processor) tenants.Processor
	ByIdProviderFunc               func(id uuid.UUID) model.Provider[tenants.RestModel]
	ByRegionAndVersionProviderFunc func(region string, majorVersion uint16, minorVersion uint16) model.Provider[tenants.RestModel]
	AllProviderFunc                func(page model.Page) model.Provider[model.Paged[tenants.RestModel]]
	ViewByIdProviderFunc           func(id uuid.UUID) model.Provider[tenants.ViewRestModel]
	AllViewProviderFunc            func(page model.Page) model.Provider[model.Paged[tenants.ViewRestModel]]
	GetByIdFunc                    func(id uuid.UUID) (tenants.RestModel, error)
	GetByRegionAndVersionFunc      func(region string, majorVersion uint16, minorVersion uint16) (tenants.RestModel, error)
	UpdateByIdFunc                 func(tenantId uuid.UUID, input tenants.RestModel) error
	DeleteByIdFunc                 func(tenantId uuid.UUID) error
	CreateFunc                     func(input tenants.RestModel) (uuid.UUID, error)
	ResetByIdFunc                  func(tenantId uuid.UUID, sections []string) (tenants.ViewRestModel, error)
}

var _ tenants.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) WithValidator(v *preset.Validator) tenants.Processor {
	if m.WithValidatorFunc != nil {
		return m.WithValidatorFunc(v)
	}
	return m
}

func (m *ProcessorMock) WithTemplates(tp templates.Processor) tenants.Processor {
	if m.WithTemplatesFunc != nil {
		return m.WithTemplatesFunc(tp)
	}
	return m
}

func (m *ProcessorMock) ViewByIdProvider(id uuid.UUID) model.Provider[tenants.ViewRestModel] {
	if m.ViewByIdProviderFunc != nil {
		return m.ViewByIdProviderFunc(id)
	}
	return model.FixedProvider(tenants.ViewRestModel{})
}

func (m *ProcessorMock) AllViewProvider(page model.Page) model.Provider[model.Paged[tenants.ViewRestModel]] {
	if m.AllViewProviderFunc != nil {
		return m.AllViewProviderFunc(page)
	}
	return model.FixedProvider(model.Paged[tenants.ViewRestModel]{})
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

func (m *ProcessorMock) AllProvider(page model.Page) model.Provider[model.Paged[tenants.RestModel]] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc(page)
	}
	return model.FixedProvider(model.Paged[tenants.RestModel]{})
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

func (m *ProcessorMock) ResetById(tenantId uuid.UUID, sections []string) (tenants.ViewRestModel, error) {
	if m.ResetByIdFunc != nil {
		return m.ResetByIdFunc(tenantId, sections)
	}
	return tenants.ViewRestModel{}, nil
}
