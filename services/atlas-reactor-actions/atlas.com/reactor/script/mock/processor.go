package mock

import (
	"time"

	"atlas-reactor-actions/script"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	ByIdProviderFunc        func(id uuid.UUID) model.Provider[script.ReactorScript]
	ByReactorIdProviderFunc func(reactorId string) model.Provider[script.ReactorScript]
	AllProviderFunc         func() model.Provider[[]script.ReactorScript]
	CreateFunc              func(m script.ReactorScript) (script.ReactorScript, error)
	UpdateFunc              func(id uuid.UUID, m script.ReactorScript) (script.ReactorScript, error)
	DeleteFunc              func(id uuid.UUID) error
	ProcessHitFunc          func(reactorId string, reactorState int8, characterId uint32) script.ProcessResult
	ProcessTriggerFunc      func(reactorId string, reactorState int8, characterId uint32) script.ProcessResult
	CountFunc               func() (int64, *time.Time, error)
}

var _ script.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByIdProvider(id uuid.UUID) model.Provider[script.ReactorScript] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return model.FixedProvider(script.ReactorScript{})
}

func (m *ProcessorMock) ByReactorIdProvider(reactorId string) model.Provider[script.ReactorScript] {
	if m.ByReactorIdProviderFunc != nil {
		return m.ByReactorIdProviderFunc(reactorId)
	}
	return model.FixedProvider(script.ReactorScript{})
}

func (m *ProcessorMock) AllProvider() model.Provider[[]script.ReactorScript] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc()
	}
	return model.FixedProvider([]script.ReactorScript{})
}

func (m *ProcessorMock) Create(ms script.ReactorScript) (script.ReactorScript, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ms)
	}
	return script.ReactorScript{}, nil
}

func (m *ProcessorMock) Update(id uuid.UUID, ms script.ReactorScript) (script.ReactorScript, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(id, ms)
	}
	return script.ReactorScript{}, nil
}

func (m *ProcessorMock) Delete(id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *ProcessorMock) ProcessHit(reactorId string, reactorState int8, characterId uint32) script.ProcessResult {
	if m.ProcessHitFunc != nil {
		return m.ProcessHitFunc(reactorId, reactorState, characterId)
	}
	return script.ProcessResult{}
}

func (m *ProcessorMock) ProcessTrigger(reactorId string, reactorState int8, characterId uint32) script.ProcessResult {
	if m.ProcessTriggerFunc != nil {
		return m.ProcessTriggerFunc(reactorId, reactorState, characterId)
	}
	return script.ProcessResult{}
}

func (m *ProcessorMock) Count() (int64, *time.Time, error) {
	if m.CountFunc != nil {
		return m.CountFunc()
	}
	return 0, nil, nil
}
