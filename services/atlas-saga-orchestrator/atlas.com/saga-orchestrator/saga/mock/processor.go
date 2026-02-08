package mock

import (
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/compartment"
	"atlas-saga-orchestrator/guild"
	"atlas-saga-orchestrator/invite"
	"atlas-saga-orchestrator/saga"
	"atlas-saga-orchestrator/skill"
	"atlas-saga-orchestrator/validation"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

// ProcessorMock is a mock implementation of the saga.Processor interface
type ProcessorMock struct {
	WithCharacterProcessorFunc   func(character.Processor) saga.Processor
	WithCompartmentProcessorFunc func(compartment.Processor) saga.Processor
	WithSkillProcessorFunc       func(skill.Processor) saga.Processor
	WithValidationProcessorFunc  func(validation.Processor) saga.Processor
	WithGuildProcessorFunc       func(guild.Processor) saga.Processor
	WithInviteProcessorFunc      func(invite.Processor) saga.Processor

	GetAllFunc      func() ([]saga.Saga, error)
	AllProviderFunc func() model.Provider[[]saga.Saga]
	GetByIdFunc     func(transactionId uuid.UUID) (saga.Saga, error)
	ByIdProviderFunc func(transactionId uuid.UUID) model.Provider[saga.Saga]

	PutFunc                             func(s saga.Saga) error
	MarkFurthestCompletedStepFailedFunc func(transactionId uuid.UUID) error
	MarkEarliestPendingStepFunc         func(transactionId uuid.UUID, status saga.Status) error
	MarkEarliestPendingStepCompletedFunc func(transactionId uuid.UUID) error
	StepCompletedFunc                   func(transactionId uuid.UUID, success bool) error
	StepCompletedWithResultFunc         func(transactionId uuid.UUID, success bool, result map[string]any) error
	AddStepFunc                         func(transactionId uuid.UUID, step saga.Step[any]) error
	AddStepAfterCurrentFunc             func(transactionId uuid.UUID, step saga.Step[any]) error
	StepFunc                            func(transactionId uuid.UUID) error
}

// WithCharacterProcessor is a mock implementation
func (m *ProcessorMock) WithCharacterProcessor(p character.Processor) saga.Processor {
	if m.WithCharacterProcessorFunc != nil {
		return m.WithCharacterProcessorFunc(p)
	}
	return m
}

// WithCompartmentProcessor is a mock implementation
func (m *ProcessorMock) WithCompartmentProcessor(p compartment.Processor) saga.Processor {
	if m.WithCompartmentProcessorFunc != nil {
		return m.WithCompartmentProcessorFunc(p)
	}
	return m
}

// WithSkillProcessor is a mock implementation
func (m *ProcessorMock) WithSkillProcessor(p skill.Processor) saga.Processor {
	if m.WithSkillProcessorFunc != nil {
		return m.WithSkillProcessorFunc(p)
	}
	return m
}

// WithValidationProcessor is a mock implementation
func (m *ProcessorMock) WithValidationProcessor(p validation.Processor) saga.Processor {
	if m.WithValidationProcessorFunc != nil {
		return m.WithValidationProcessorFunc(p)
	}
	return m
}

// WithGuildProcessor is a mock implementation
func (m *ProcessorMock) WithGuildProcessor(p guild.Processor) saga.Processor {
	if m.WithGuildProcessorFunc != nil {
		return m.WithGuildProcessorFunc(p)
	}
	return m
}

// WithInviteProcessor is a mock implementation
func (m *ProcessorMock) WithInviteProcessor(p invite.Processor) saga.Processor {
	if m.WithInviteProcessorFunc != nil {
		return m.WithInviteProcessorFunc(p)
	}
	return m
}

// GetAll is a mock implementation
func (m *ProcessorMock) GetAll() ([]saga.Saga, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return nil, nil
}

// AllProvider is a mock implementation
func (m *ProcessorMock) AllProvider() model.Provider[[]saga.Saga] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc()
	}
	return model.FixedProvider[[]saga.Saga](nil)
}

// GetById is a mock implementation
func (m *ProcessorMock) GetById(transactionId uuid.UUID) (saga.Saga, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(transactionId)
	}
	return saga.Saga{}, nil
}

// ByIdProvider is a mock implementation
func (m *ProcessorMock) ByIdProvider(transactionId uuid.UUID) model.Provider[saga.Saga] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(transactionId)
	}
	return model.FixedProvider[saga.Saga](saga.Saga{})
}

// Put is a mock implementation
func (m *ProcessorMock) Put(s saga.Saga) error {
	if m.PutFunc != nil {
		return m.PutFunc(s)
	}
	return nil
}

// MarkFurthestCompletedStepFailed is a mock implementation
func (m *ProcessorMock) MarkFurthestCompletedStepFailed(transactionId uuid.UUID) error {
	if m.MarkFurthestCompletedStepFailedFunc != nil {
		return m.MarkFurthestCompletedStepFailedFunc(transactionId)
	}
	return nil
}

// MarkEarliestPendingStep is a mock implementation
func (m *ProcessorMock) MarkEarliestPendingStep(transactionId uuid.UUID, status saga.Status) error {
	if m.MarkEarliestPendingStepFunc != nil {
		return m.MarkEarliestPendingStepFunc(transactionId, status)
	}
	return nil
}

// MarkEarliestPendingStepCompleted is a mock implementation
func (m *ProcessorMock) MarkEarliestPendingStepCompleted(transactionId uuid.UUID) error {
	if m.MarkEarliestPendingStepCompletedFunc != nil {
		return m.MarkEarliestPendingStepCompletedFunc(transactionId)
	}
	return nil
}

// StepCompleted is a mock implementation
func (m *ProcessorMock) StepCompleted(transactionId uuid.UUID, success bool) error {
	if m.StepCompletedFunc != nil {
		return m.StepCompletedFunc(transactionId, success)
	}
	return nil
}

// StepCompletedWithResult is a mock implementation
func (m *ProcessorMock) StepCompletedWithResult(transactionId uuid.UUID, success bool, result map[string]any) error {
	if m.StepCompletedWithResultFunc != nil {
		return m.StepCompletedWithResultFunc(transactionId, success, result)
	}
	return nil
}

// AddStep is a mock implementation
func (m *ProcessorMock) AddStep(transactionId uuid.UUID, step saga.Step[any]) error {
	if m.AddStepFunc != nil {
		return m.AddStepFunc(transactionId, step)
	}
	return nil
}

// AddStepAfterCurrent is a mock implementation
func (m *ProcessorMock) AddStepAfterCurrent(transactionId uuid.UUID, step saga.Step[any]) error {
	if m.AddStepAfterCurrentFunc != nil {
		return m.AddStepAfterCurrentFunc(transactionId, step)
	}
	return nil
}

// Step is a mock implementation
func (m *ProcessorMock) Step(transactionId uuid.UUID) error {
	if m.StepFunc != nil {
		return m.StepFunc(transactionId)
	}
	return nil
}
