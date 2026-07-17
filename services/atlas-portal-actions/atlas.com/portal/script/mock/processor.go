package mock

import (
	"atlas-portal-actions/script"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	CreateFunc             func(m script.PortalScript) (script.PortalScript, error)
	UpdateFunc             func(id uuid.UUID, m script.PortalScript) (script.PortalScript, error)
	DeleteFunc             func(id uuid.UUID) error
	ByIdProviderFunc       func(id uuid.UUID) model.Provider[script.PortalScript]
	ByPortalIdProviderFunc func(portalId string) model.Provider[script.PortalScript]
	AllProviderFunc        func() model.Provider[[]script.PortalScript]
	CountFunc              func() (int64, *time.Time, error)
	ProcessFunc            func(f field.Model, characterId uint32, portalName string, portalId uint32) script.ProcessResult
}

var _ script.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Create(ms script.PortalScript) (script.PortalScript, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ms)
	}
	return script.PortalScript{}, nil
}

func (m *ProcessorMock) Update(id uuid.UUID, ms script.PortalScript) (script.PortalScript, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(id, ms)
	}
	return script.PortalScript{}, nil
}

func (m *ProcessorMock) Delete(id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *ProcessorMock) ByIdProvider(id uuid.UUID) model.Provider[script.PortalScript] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return model.FixedProvider(script.PortalScript{})
}

func (m *ProcessorMock) ByPortalIdProvider(portalId string) model.Provider[script.PortalScript] {
	if m.ByPortalIdProviderFunc != nil {
		return m.ByPortalIdProviderFunc(portalId)
	}
	return model.FixedProvider(script.PortalScript{})
}

func (m *ProcessorMock) AllProvider() model.Provider[[]script.PortalScript] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc()
	}
	return model.FixedProvider([]script.PortalScript{})
}

func (m *ProcessorMock) Count() (int64, *time.Time, error) {
	if m.CountFunc != nil {
		return m.CountFunc()
	}
	return 0, nil, nil
}

func (m *ProcessorMock) Process(f field.Model, characterId uint32, portalName string, portalId uint32) script.ProcessResult {
	if m.ProcessFunc != nil {
		return m.ProcessFunc(f, characterId, portalName, portalId)
	}
	return script.ProcessResult{}
}
