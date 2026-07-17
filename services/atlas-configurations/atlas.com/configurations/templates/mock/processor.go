package mock

import (
	"atlas-configurations/templates"
	"atlas-configurations/templates/characters/preset"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	WithValidatorFunc              func(v *preset.Validator) templates.Processor
	ByRegionAndVersionProviderFunc func(region string, majorVersion uint16, minorVersion uint16) model.Provider[templates.RestModel]
	ByIdProviderFunc               func(templateId uuid.UUID) model.Provider[templates.RestModel]
	AllProviderFunc                func(page model.Page) model.Provider[model.Paged[templates.RestModel]]
	GetByRegionAndVersionFunc      func(region string, majorVersion uint16, minorVersion uint16) (templates.RestModel, error)
	GetByIdFunc                    func(templateId uuid.UUID) (templates.RestModel, error)
	CreateFunc                     func(input templates.RestModel) (uuid.UUID, error)
	UpdateByIdFunc                 func(templateId uuid.UUID, input templates.RestModel) error
	DeleteByIdFunc                 func(templateId uuid.UUID) error
}

var _ templates.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) WithValidator(v *preset.Validator) templates.Processor {
	if m.WithValidatorFunc != nil {
		return m.WithValidatorFunc(v)
	}
	return m
}

func (m *ProcessorMock) ByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[templates.RestModel] {
	if m.ByRegionAndVersionProviderFunc != nil {
		return m.ByRegionAndVersionProviderFunc(region, majorVersion, minorVersion)
	}
	return model.FixedProvider(templates.RestModel{})
}

func (m *ProcessorMock) ByIdProvider(templateId uuid.UUID) model.Provider[templates.RestModel] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(templateId)
	}
	return model.FixedProvider(templates.RestModel{})
}

func (m *ProcessorMock) AllProvider(page model.Page) model.Provider[model.Paged[templates.RestModel]] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc(page)
	}
	return model.FixedProvider(model.Paged[templates.RestModel]{})
}

func (m *ProcessorMock) GetByRegionAndVersion(region string, majorVersion uint16, minorVersion uint16) (templates.RestModel, error) {
	if m.GetByRegionAndVersionFunc != nil {
		return m.GetByRegionAndVersionFunc(region, majorVersion, minorVersion)
	}
	return templates.RestModel{}, nil
}

func (m *ProcessorMock) GetById(templateId uuid.UUID) (templates.RestModel, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(templateId)
	}
	return templates.RestModel{}, nil
}

func (m *ProcessorMock) Create(input templates.RestModel) (uuid.UUID, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(input)
	}
	return uuid.Nil, nil
}

func (m *ProcessorMock) UpdateById(templateId uuid.UUID, input templates.RestModel) error {
	if m.UpdateByIdFunc != nil {
		return m.UpdateByIdFunc(templateId, input)
	}
	return nil
}

func (m *ProcessorMock) DeleteById(templateId uuid.UUID) error {
	if m.DeleteByIdFunc != nil {
		return m.DeleteByIdFunc(templateId)
	}
	return nil
}
