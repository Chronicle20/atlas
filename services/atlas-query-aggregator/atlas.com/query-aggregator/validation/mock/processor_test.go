package mock

import (
	"atlas-query-aggregator/validation"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"testing"
)

func TestProcessor_ValidateStructured(t *testing.T) {
	t.Run("Default behavior", func(t *testing.T) {
		processor := &ProcessorImpl{}

		// Call ValidateStructured with no custom function
		result, err := processor.ValidateStructured()(123, []validation.ConditionInput{})

		// Check that there's no error
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check that the result has the correct character ID
		if result.CharacterId() != 123 {
			t.Errorf("Expected character ID 123, got %d", result.CharacterId())
		}

		// Check that the result passed (default behavior)
		if !result.Passed() {
			t.Errorf("Expected result to pass, but it failed")
		}
	})

	t.Run("Custom behavior - success", func(t *testing.T) {
		processor := &ProcessorImpl{
			ValidateStructuredFunc: func(decorators ...model.Decorator[validation.ValidationResult]) func(characterId uint32, conditionInputs []validation.ConditionInput) (validation.ValidationResult, error) {
				return func(characterId uint32, conditionInputs []validation.ConditionInput) (validation.ValidationResult, error) {
					result := validation.NewValidationResult(characterId)
					condResult := validation.ConditionResult{
						Passed:      true,
						Description: "Custom condition",
						Type:        validation.JobCondition,
						Operator:    validation.Equals,
						Value:       100,
						ActualValue: 100,
					}
					result.AddConditionResult(condResult)
					return result, nil
				}
			},
		}

		// Call ValidateStructured with custom function
		result, err := processor.ValidateStructured()(123, []validation.ConditionInput{
			{Type: "jobId", Operator: "=", Value: 100},
		})

		// Check that there's no error
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check that the result has the correct character ID
		if result.CharacterId() != 123 {
			t.Errorf("Expected character ID 123, got %d", result.CharacterId())
		}

		// Check that the result passed
		if !result.Passed() {
			t.Errorf("Expected result to pass, but it failed")
		}

		// Check that the details contain our custom message
		if len(result.Details()) != 1 || result.Details()[0] != "Passed: Custom condition" {
			t.Errorf("Expected details to contain 'Passed: Custom condition', got %v", result.Details())
		}
	})

	t.Run("Custom behavior - failure", func(t *testing.T) {
		processor := &ProcessorImpl{
			ValidateStructuredFunc: func(decorators ...model.Decorator[validation.ValidationResult]) func(characterId uint32, conditionInputs []validation.ConditionInput) (validation.ValidationResult, error) {
				return func(characterId uint32, conditionInputs []validation.ConditionInput) (validation.ValidationResult, error) {
					return validation.ValidationResult{}, errors.New("custom error")
				}
			},
		}

		// Call ValidateStructured with custom function
		_, err := processor.ValidateStructured()(123, []validation.ConditionInput{
			{Type: "jobId", Operator: "=", Value: 100},
		})

		// Check that there's an error
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		// Check the error message
		if err.Error() != "custom error" {
			t.Errorf("Expected error message 'custom error', got '%v'", err.Error())
		}
	})
}
