package mock

import (
	"atlas-saga-orchestrator/validation"
)

// ProcessorMock is a mock implementation of the validation.Processor interface
type ProcessorMock struct {
	// ValidateCharacterStateFunc is a function field for the ValidateCharacterState method
	ValidateCharacterStateFunc func(characterId uint32, conditions []validation.ConditionInput) (validation.ValidationResult, error)
}

// ValidateCharacterState is a mock implementation of the validation.Processor.ValidateCharacterState method
func (m *ProcessorMock) ValidateCharacterState(characterId uint32, conditions []validation.ConditionInput) (validation.ValidationResult, error) {
	if m.ValidateCharacterStateFunc != nil {
		return m.ValidateCharacterStateFunc(characterId, conditions)
	}
	// Default implementation returns an empty validation result
	return validation.NewValidationResult(characterId), nil
}