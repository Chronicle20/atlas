package mock

import (
	"atlas-query-aggregator/validation"
	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorImpl is a mock implementation of the validation.ProcessorImpl
type ProcessorImpl struct {
	ValidateFunc func(decorators ...model.Decorator[validation.ValidationResult]) func(characterId uint32, conditionExpressions []string) (validation.ValidationResult, error)
}

// Validate returns a function that validates conditions against a character
func (m *ProcessorImpl) Validate(decorators ...model.Decorator[validation.ValidationResult]) func(characterId uint32, conditionExpressions []string) (validation.ValidationResult, error) {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(decorators...)
	}
	return func(characterId uint32, conditionExpressions []string) (validation.ValidationResult, error) {
		return validation.NewValidationResult(characterId), nil
	}
}
