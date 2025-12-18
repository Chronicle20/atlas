package validation

import (
	"context"
	"github.com/sirupsen/logrus"
)

// Processor is the interface for validation operations
type Processor interface {
	// ValidateCharacterState validates a character's state against a set of conditions
	ValidateCharacterState(characterId uint32, conditions []ConditionInput) (ValidationResult, error)
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new validation processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// ValidateCharacterState validates a character's state against a set of conditions
func (p *ProcessorImpl) ValidateCharacterState(characterId uint32, conditions []ConditionInput) (ValidationResult, error) {
	// Create the request body
	requestBody := RestModel{
		Id:         characterId,
		Conditions: conditions,
	}

	// Create and execute the HTTP request
	resp, err := requestById(characterId, requestBody)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).WithFields(logrus.Fields{
			"character_id": characterId,
		}).Error("Failed to execute validation request")
		return ValidationResult{}, err
	}

	// Create the validation result
	result := NewValidationResult(characterId)

	// Add the condition results
	for _, condResult := range resp.Results {
		result.AddConditionResult(condResult)
	}

	// Set the overall result based on the response
	if !resp.Passed {
		// If the validation failed, return the result with details
		return result, nil
	}

	return result, nil
}
