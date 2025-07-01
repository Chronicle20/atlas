package validation

import (
	"atlas-query-aggregator/character"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Validate(decorators ...model.Decorator[ValidationResult]) func(characterId uint32, conditionExpressions []string) (ValidationResult, error)
}

// ProcessorImpl handles validation logic
type ProcessorImpl struct {
	l                  logrus.FieldLogger
	ctx                context.Context
	characterProcessor character.Processor
}

// NewProcessor creates a new validation processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:                  l,
		ctx:                ctx,
		characterProcessor: character.NewProcessor(l, ctx),
	}
}

// Validate validates a list of conditions against a character
func (p *ProcessorImpl) Validate(decorators ...model.Decorator[ValidationResult]) func(characterId uint32, conditionExpressions []string) (ValidationResult, error) {
	return func(characterId uint32, conditionExpressions []string) (ValidationResult, error) {
		// Create a new validation result
		result := NewValidationResult(characterId)

		// Get character data
		characterData, err := p.characterProcessor.GetById()(characterId)
		if err != nil {
			return result, fmt.Errorf("failed to get character data: %w", err)
		}

		// Parse and evaluate each condition
		for _, expr := range conditionExpressions {
			condition, err := NewCondition(expr)
			if err != nil {
				return result, fmt.Errorf("invalid condition: %w", err)
			}

			// Evaluate the condition
			passed, description := condition.Evaluate(characterData)
			result.AddResult(passed, description)
		}

		// Apply decorators
		return model.Map(model.Decorate(decorators))(func() (ValidationResult, error) {
			return result, nil
		})()
	}
}
