package mock

import (
	"atlas-map-actions/validation"
)

type ProcessorMock struct {
	ValidateCharacterStateFunc func(characterId uint32, conditions []validation.ConditionInput) (validation.ValidationResult, error)
}

var _ validation.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ValidateCharacterState(characterId uint32, conditions []validation.ConditionInput) (validation.ValidationResult, error) {
	if m.ValidateCharacterStateFunc != nil {
		return m.ValidateCharacterStateFunc(characterId, conditions)
	}
	return validation.ValidationResult{}, nil
}
