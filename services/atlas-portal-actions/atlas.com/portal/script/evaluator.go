package script

import (
	"context"
	"fmt"
	"strconv"

	"atlas-portal-actions/validation"

	"github.com/Chronicle20/atlas-script-core/condition"
	scriptctx "github.com/Chronicle20/atlas-script-core/context"
	"github.com/sirupsen/logrus"
)

// ConditionEvaluator evaluates conditions for portal scripts
type ConditionEvaluator struct {
	l           logrus.FieldLogger
	ctx         context.Context
	validationP validation.Processor
}

// NewConditionEvaluator creates a new condition evaluator
func NewConditionEvaluator(l logrus.FieldLogger, ctx context.Context) *ConditionEvaluator {
	return &ConditionEvaluator{
		l:           l,
		ctx:         ctx,
		validationP: validation.NewProcessor(l, ctx),
	}
}

// EvaluateCondition evaluates a single condition for a character
func (e *ConditionEvaluator) EvaluateCondition(characterId uint32, cond condition.Model) (bool, error) {
	e.l.Debugf("Evaluating condition [%s] for character [%d]", cond.Type(), characterId)

	// Evaluate the value (support arithmetic expressions)
	valueStr := cond.Value()
	intValue, err := scriptctx.EvaluateValueAsInt(valueStr)
	if err != nil {
		e.l.WithError(err).Errorf("Failed to evaluate value [%s] for condition", valueStr)
		return false, err
	}

	// Resolve referenceId if present
	var resolvedReferenceId uint32
	referenceIdRaw := cond.ReferenceIdRaw()
	if referenceIdRaw != "" {
		if refIdInt, convErr := strconv.ParseUint(referenceIdRaw, 10, 32); convErr == nil {
			resolvedReferenceId = uint32(refIdInt)
		} else {
			e.l.Errorf("ReferenceId [%s] is not a valid uint32 for character [%d]", referenceIdRaw, characterId)
			return false, fmt.Errorf("referenceId [%s] is not a valid uint32", referenceIdRaw)
		}
	}

	// Create validation condition input
	validationCondition := validation.ConditionInput{
		Type:            cond.Type(),
		Operator:        cond.Operator(),
		Value:           intValue,
		ReferenceId:     resolvedReferenceId,
		Step:            cond.Step(),
		IncludeEquipped: cond.IncludeEquipped(),
	}

	// Validate the character state using the validation processor
	result, err := e.validationP.ValidateCharacterState(characterId, []validation.ConditionInput{validationCondition})
	if err != nil {
		e.l.WithError(err).Errorf("Failed to validate character state for condition [%s]", cond.Type())
		return false, err
	}

	e.l.Debugf("Condition [%s] evaluated to [%t] for character [%d]", cond.Type(), result.Passed(), characterId)
	return result.Passed(), nil
}
