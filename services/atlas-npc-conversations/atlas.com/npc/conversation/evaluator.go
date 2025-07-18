package conversation

import (
	"atlas-npc-conversations/validation"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

// Evaluator is the interface for evaluating conditions in conversations
type Evaluator interface {
	// EvaluateCondition evaluates a condition for a character
	EvaluateCondition(characterId uint32, condition ConditionModel) (bool, error)
}

// EvaluatorImpl is the implementation of the Evaluator interface
type EvaluatorImpl struct {
	l           logrus.FieldLogger
	ctx         context.Context
	validationP validation.Processor
	t           tenant.Model
}

// NewEvaluator creates a new condition evaluator
func NewEvaluator(l logrus.FieldLogger, ctx context.Context, t tenant.Model) Evaluator {
	return &EvaluatorImpl{
		l:           l,
		ctx:         ctx,
		validationP: validation.NewProcessor(l, ctx),
		t:           t,
	}
}

// EvaluateCondition evaluates a condition for a character
func (e *EvaluatorImpl) EvaluateCondition(characterId uint32, condition ConditionModel) (bool, error) {
	e.l.Debugf("Evaluating condition [%s] for character [%d]", condition.Type(), characterId)

	// Get the conversation context
	ctx, err := GetRegistry().GetPreviousContext(e.t, characterId)
	if err != nil {
		e.l.WithError(err).Errorf("Failed to get conversation context for character [%d]", characterId)
		return false, err
	}

	// Get the value from the condition
	valueStr := condition.Value()
	var value int

	// Check if the value is a context reference
	if strings.HasPrefix(valueStr, "context.") {
		// Extract the context key
		contextKey := strings.TrimPrefix(valueStr, "context.")

		// Look up the value in the context map
		contextValue, exists := ctx.Context()[contextKey]
		if !exists {
			e.l.Errorf("Context key [%s] not found in conversation context", contextKey)
			return false, fmt.Errorf("context key [%s] not found", contextKey)
		}

		// Convert the context value to an integer
		var err error
		value, err = strconv.Atoi(contextValue)
		if err != nil {
			e.l.WithError(err).Errorf("Failed to convert context value [%s] to integer", contextValue)
			return false, fmt.Errorf("context value [%s] is not a valid integer", contextValue)
		}
	} else {
		// Try to convert the value directly to an integer
		var err error
		value, err = strconv.Atoi(valueStr)
		if err != nil {
			e.l.WithError(err).Errorf("Failed to convert value [%s] to integer", valueStr)
			return false, fmt.Errorf("value [%s] is not a valid integer", valueStr)
		}
	}

	// Create a validation condition input
	validationCondition := validation.ConditionInput{
		Type:     condition.Type(),
		Operator: condition.Operator(),
		Value:    value,
		ItemId:   condition.ItemId(),
	}

	// Validate the character state using the validation processor
	result, err := e.validationP.ValidateCharacterState(characterId, []validation.ConditionInput{validationCondition})
	if err != nil {
		e.l.WithError(err).Errorf("Failed to validate character state for condition [%+v]", condition)
		return false, err
	}

	e.l.Debugf("Condition [%s] evaluated to [%t] for character [%d]. Operator [%s], Value [%d].", condition.Type(), result.Passed(), characterId, condition.Operator(), value)
	return result.Passed(), nil
}
