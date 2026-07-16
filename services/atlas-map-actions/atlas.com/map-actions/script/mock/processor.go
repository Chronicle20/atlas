package mock

import (
	"time"

	"atlas-map-actions/script"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	CreateFunc                      func(m script.MapScript) (script.MapScript, error)
	UpdateFunc                      func(id uuid.UUID, m script.MapScript) (script.MapScript, error)
	DeleteFunc                      func(id uuid.UUID) error
	ByIdProviderFunc                func(id uuid.UUID) model.Provider[script.MapScript]
	ByScriptNameProviderFunc        func(scriptName string, page model.Page) model.Provider[model.Paged[script.MapScript]]
	ByScriptNameAndTypeProviderFunc func(scriptName string, scriptType string) model.Provider[script.MapScript]
	AllProviderFunc                 func(page model.Page) model.Provider[model.Paged[script.MapScript]]
	DeleteAllForTenantFunc          func() (int64, error)
	CountFunc                       func() (int64, *time.Time, error)
	ProcessFunc                     func(f field.Model, characterId uint32, scriptName string, scriptType string) script.ProcessResult
}

var _ script.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Create(ms script.MapScript) (script.MapScript, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ms)
	}
	return script.MapScript{}, nil
}

func (m *ProcessorMock) Update(id uuid.UUID, ms script.MapScript) (script.MapScript, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(id, ms)
	}
	return script.MapScript{}, nil
}

func (m *ProcessorMock) Delete(id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *ProcessorMock) ByIdProvider(id uuid.UUID) model.Provider[script.MapScript] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return model.FixedProvider(script.MapScript{})
}

func (m *ProcessorMock) ByScriptNameProvider(scriptName string, page model.Page) model.Provider[model.Paged[script.MapScript]] {
	if m.ByScriptNameProviderFunc != nil {
		return m.ByScriptNameProviderFunc(scriptName, page)
	}
	return model.FixedProvider(model.Paged[script.MapScript]{})
}

func (m *ProcessorMock) ByScriptNameAndTypeProvider(scriptName string, scriptType string) model.Provider[script.MapScript] {
	if m.ByScriptNameAndTypeProviderFunc != nil {
		return m.ByScriptNameAndTypeProviderFunc(scriptName, scriptType)
	}
	return model.FixedProvider(script.MapScript{})
}

func (m *ProcessorMock) AllProvider(page model.Page) model.Provider[model.Paged[script.MapScript]] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc(page)
	}
	return model.FixedProvider(model.Paged[script.MapScript]{})
}

func (m *ProcessorMock) DeleteAllForTenant() (int64, error) {
	if m.DeleteAllForTenantFunc != nil {
		return m.DeleteAllForTenantFunc()
	}
	return 0, nil
}

func (m *ProcessorMock) Count() (int64, *time.Time, error) {
	if m.CountFunc != nil {
		return m.CountFunc()
	}
	return 0, nil, nil
}

func (m *ProcessorMock) Process(f field.Model, characterId uint32, scriptName string, scriptType string) script.ProcessResult {
	if m.ProcessFunc != nil {
		return m.ProcessFunc(f, characterId, scriptName, scriptType)
	}
	return script.ProcessResult{}
}
