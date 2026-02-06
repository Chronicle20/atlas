package mock

import (
	"atlas-query-aggregator/validation"

	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorImpl is a mock implementation of the validation.ProcessorImpl
type ProcessorImpl struct {
	ValidateStructuredFunc  func(decorators ...model.Decorator[validation.ValidationResult]) func(characterId uint32, conditionInputs []validation.ConditionInput) (validation.ValidationResult, error)
	ValidateWithContextFunc func(decorators ...model.Decorator[validation.ValidationResult]) func(ctx validation.ValidationContext, conditionInputs []validation.ConditionInput) (validation.ValidationResult, error)
}

// ValidateStructured returns a function that validates structured conditions against a character
func (m *ProcessorImpl) ValidateStructured(decorators ...model.Decorator[validation.ValidationResult]) func(characterId uint32, conditionInputs []validation.ConditionInput) (validation.ValidationResult, error) {
	if m.ValidateStructuredFunc != nil {
		return m.ValidateStructuredFunc(decorators...)
	}
	return func(characterId uint32, conditionInputs []validation.ConditionInput) (validation.ValidationResult, error) {
		return validation.NewValidationResult(characterId), nil
	}
}

// ValidateWithContext returns a function that validates conditions using a validation context
func (m *ProcessorImpl) ValidateWithContext(decorators ...model.Decorator[validation.ValidationResult]) func(ctx validation.ValidationContext, conditionInputs []validation.ConditionInput) (validation.ValidationResult, error) {
	if m.ValidateWithContextFunc != nil {
		return m.ValidateWithContextFunc(decorators...)
	}
	return func(ctx validation.ValidationContext, conditionInputs []validation.ConditionInput) (validation.ValidationResult, error) {
		return validation.NewValidationResult(ctx.Character().Id()), nil
	}
}
