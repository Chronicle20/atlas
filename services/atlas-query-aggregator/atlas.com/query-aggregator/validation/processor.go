package validation

import (
	"atlas-query-aggregator/character"
	"atlas-query-aggregator/inventory"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	// ValidateStructured validates a list of structured condition inputs against a character
	ValidateStructured(decorators ...model.Decorator[ValidationResult]) func(characterId uint32, conditionInputs []ConditionInput) (ValidationResult, error)
}

// ProcessorImpl handles validation logic
type ProcessorImpl struct {
	l                  logrus.FieldLogger
	ctx                context.Context
	characterProcessor character.Processor
	inventoryProcessor inventory.Processor
}

// NewProcessor creates a new validation processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:                  l,
		ctx:                ctx,
		characterProcessor: character.NewProcessor(l, ctx),
		inventoryProcessor: inventory.NewProcessor(l, ctx),
	}
}


// ValidateStructured validates a list of structured condition inputs against a character
func (p *ProcessorImpl) ValidateStructured(decorators ...model.Decorator[ValidationResult]) func(characterId uint32, conditionInputs []ConditionInput) (ValidationResult, error) {
	return func(characterId uint32, conditionInputs []ConditionInput) (ValidationResult, error) {
		// Create a new validation result
		result := NewValidationResult(characterId)

		// Parse all conditions
		conditions := make([]Condition, 0, len(conditionInputs))
		needsInventory := false

		for _, input := range conditionInputs {
			condition, err := NewConditionBuilder().FromInput(input).Build()
			if err != nil {
				return result, fmt.Errorf("invalid condition: %w", err)
			}

			conditions = append(conditions, condition)

			// Check if this condition requires inventory data
			if condition.conditionType == ItemCondition {
				needsInventory = true
			}
		}

		// Get character data with inventory if needed
		var characterData character.Model
		var err error

		if needsInventory {
			// Use the InventoryDecorator to ensure the character has inventory data
			characterData, err = p.characterProcessor.GetById(p.characterProcessor.InventoryDecorator)(characterId)
		} else {
			characterData, err = p.characterProcessor.GetById()(characterId)
		}

		if err != nil {
			return result, fmt.Errorf("failed to get character data: %w", err)
		}

		// Evaluate each condition
		for _, condition := range conditions {
			conditionResult := condition.Evaluate(characterData)
			result.AddConditionResult(conditionResult)
		}

		// Apply decorators
		return model.Map(model.Decorate(decorators))(func() (ValidationResult, error) {
			return result, nil
		})()
	}
}
