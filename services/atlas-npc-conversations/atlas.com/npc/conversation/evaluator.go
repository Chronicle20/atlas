package conversation

import (
	"atlas-npc-conversations/validation"
	"context"
	"fmt"
	"strconv"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	scriptctx "github.com/Chronicle20/atlas-script-core/context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
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
	ctx, err := GetRegistry().GetPreviousContext(e.ctx, characterId)
	if err != nil {
		e.l.WithError(err).Errorf("Failed to get conversation context for character [%d]", characterId)
		return false, err
	}

	// Get the value from the condition
	valueStr := condition.Value()

	// Use the new ExtractContextValue function which supports both "{context.xxx}" and "context.xxx" formats
	extractedValue, isContextRef, err := ExtractContextValue(valueStr, ctx.Context())
	if err != nil {
		e.l.WithError(err).Errorf("Failed to extract context value for condition")
		return false, err
	}

	if isContextRef {
		e.l.Debugf("Resolved context reference [%s] to [%s] for character [%d]", valueStr, extractedValue, characterId)
	}

	// Evaluate arithmetic expressions if present (e.g., "10 * 5" -> 50)
	intValue, err := evaluateValueAsInt(extractedValue)
	if err != nil {
		e.l.WithError(err).Errorf("Failed to evaluate value [%s] for condition", extractedValue)
		return false, err
	}

	e.l.Debugf("Evaluated condition value [%s] to [%d] for character [%d]", extractedValue, intValue, characterId)

	// Resolve referenceId from condition (supports context references like {context.itemId})
	var resolvedReferenceId uint32
	referenceIdRaw := condition.ReferenceIdRaw()
	if referenceIdRaw != "" {
		resolvedRefIdStr, isRefIdContextRef, refIdErr := ExtractContextValue(referenceIdRaw, ctx.Context())
		if refIdErr != nil {
			e.l.WithError(refIdErr).Errorf("Failed to resolve referenceId context [%s] for character [%d]", referenceIdRaw, characterId)
			return false, refIdErr
		}
		if isRefIdContextRef {
			e.l.Debugf("Resolved referenceId [%s] to [%s] for character [%d]", referenceIdRaw, resolvedRefIdStr, characterId)
		}
		if refIdInt, convErr := strconv.ParseUint(resolvedRefIdStr, 10, 32); convErr == nil {
			resolvedReferenceId = uint32(refIdInt)
		} else {
			e.l.Errorf("ReferenceId [%s] resolved to [%s] which is not a valid uint32 for character [%d]", referenceIdRaw, resolvedRefIdStr, characterId)
			return false, fmt.Errorf("referenceId [%s] is not a valid uint32", resolvedRefIdStr)
		}
	}

	// Create a validation condition input
	validationCondition := validation.ConditionInput{
		Type:            condition.Type(),
		Operator:        condition.Operator(),
		Value:           intValue,
		ReferenceId:     resolvedReferenceId,
		Step:            condition.Step(),
		IncludeEquipped: condition.IncludeEquipped(),
	}

	// Resolve worldId from condition (supports context references like {context.worldId})
	if condition.WorldId() != "" {
		worldIdStr, _, err := ExtractContextValue(condition.WorldId(), ctx.Context())
		if err == nil {
			if worldIdInt, convErr := strconv.Atoi(worldIdStr); convErr == nil {
				validationCondition.WorldId = world.Id(worldIdInt)
			}
		}
	}

	// Resolve channelId from condition (supports context references like {context.channelId})
	if condition.ChannelId() != "" {
		channelIdStr, _, err := ExtractContextValue(condition.ChannelId(), ctx.Context())
		if err == nil {
			if channelIdInt, convErr := strconv.Atoi(channelIdStr); convErr == nil {
				validationCondition.ChannelId = channel.Id(channelIdInt)
			}
		}
	}

	// Validate the character state using the validation processor
	result, err := e.validationP.ValidateCharacterState(characterId, []validation.ConditionInput{validationCondition})
	if err != nil {
		e.l.WithError(err).Errorf("Failed to validate character state for condition [%+v]", condition)
		return false, err
	}

	e.l.Debugf("Condition [%s] evaluated to [%t] for character [%d]. Operator [%s], Value [%d].", condition.Type(), result.Passed(), characterId, condition.Operator(), intValue)
	return result.Passed(), nil
}

// evaluateValueAsInt evaluates a string value as an integer, supporting arithmetic expressions
// Uses the shared implementation from atlas-script-core/context
func evaluateValueAsInt(value string) (int, error) {
	return scriptctx.EvaluateValueAsInt(value)
}
