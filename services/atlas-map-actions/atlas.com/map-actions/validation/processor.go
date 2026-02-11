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
	requestBody := RestModel{
		Id:         characterId,
		Conditions: conditions,
	}

	resp, err := requestById(requestBody)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).WithFields(logrus.Fields{
			"character_id": characterId,
		}).Error("Failed to execute validation request")
		return ValidationResult{}, err
	}

	return NewValidationResult(characterId, resp.Passed), nil
}
