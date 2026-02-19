package script

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ConditionEvaluator evaluates conditions for reactor scripts
type ConditionEvaluator struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewConditionEvaluator creates a new condition evaluator
func NewConditionEvaluator(l logrus.FieldLogger, ctx context.Context) *ConditionEvaluator {
	return &ConditionEvaluator{
		l:   l,
		ctx: ctx,
	}
}

// EvaluateCondition evaluates a single condition
func (e *ConditionEvaluator) EvaluateCondition(reactorState int8, characterId uint32, cond condition.Model) (bool, error) {
	e.l.Debugf("Evaluating condition [%s] with operator [%s] value [%s]", cond.Type(), cond.Operator(), cond.Value())

	switch cond.Type() {
	case "reactor_state":
		return EvaluateReactorStateCondition(reactorState, cond.Operator(), cond.Value())
	case "pq_custom_data":
		return e.evaluatePqCustomDataCondition(characterId, cond)
	default:
		return false, fmt.Errorf("unknown condition type: %s", cond.Type())
	}
}

// EvaluateRule evaluates all conditions for a rule (AND logic)
func (e *ConditionEvaluator) EvaluateRule(reactorState int8, characterId uint32, rule Rule) (bool, error) {
	conditions := rule.Conditions()

	// Empty conditions = always match (default rule)
	if len(conditions) == 0 {
		return true, nil
	}

	// All conditions must pass (AND logic)
	for _, cond := range conditions {
		passed, err := e.EvaluateCondition(reactorState, characterId, cond)
		if err != nil {
			return false, fmt.Errorf("condition evaluation failed: %w", err)
		}
		if !passed {
			return false, nil
		}
	}

	return true, nil
}

// EvaluateReactorStateCondition evaluates a reactor_state condition
func EvaluateReactorStateCondition(reactorState int8, operator string, valueStr string) (bool, error) {
	var value int8
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return false, fmt.Errorf("invalid reactor_state value: %s", valueStr)
	}

	switch operator {
	case "=":
		return reactorState == value, nil
	case "!=":
		return reactorState != value, nil
	case ">":
		return reactorState > value, nil
	case "<":
		return reactorState < value, nil
	case ">=":
		return reactorState >= value, nil
	case "<=":
		return reactorState <= value, nil
	default:
		return false, fmt.Errorf("unknown operator: %s", operator)
	}
}

// evaluatePqCustomDataCondition evaluates a pq_custom_data condition.
// Uses the condition's "step" field as the custom data key name.
func (e *ConditionEvaluator) evaluatePqCustomDataCondition(characterId uint32, cond condition.Model) (bool, error) {
	key := cond.Step()
	if key == "" {
		return false, fmt.Errorf("pq_custom_data condition requires step field as custom data key")
	}

	// Query the PQ instance for this character
	pqInstance, err := e.getPqInstanceByCharacter(characterId)
	if err != nil {
		e.l.WithError(err).Debugf("Failed to get PQ instance for character %d, treating as condition not met", characterId)
		return false, nil
	}

	// Get the custom data value
	rawValue, exists := pqInstance.StageState.CustomData[key]
	if !exists {
		e.l.Debugf("PQ custom data key [%s] not found for character %d, treating as 0", key, characterId)
		return compareIntValues(0, cond.Operator(), cond.Value())
	}

	// Convert custom data value to int
	actualValue, err := strconv.Atoi(fmt.Sprintf("%v", rawValue))
	if err != nil {
		e.l.Debugf("PQ custom data key [%s] value [%v] is not numeric, treating as 0", key, rawValue)
		return compareIntValues(0, cond.Operator(), cond.Value())
	}

	return compareIntValues(actualValue, cond.Operator(), cond.Value())
}

// compareIntValues compares an actual int value against an expected value string using the operator
func compareIntValues(actual int, operator string, valueStr string) (bool, error) {
	expected, err := strconv.Atoi(valueStr)
	if err != nil {
		return false, fmt.Errorf("invalid numeric value: %s", valueStr)
	}

	switch operator {
	case "=":
		return actual == expected, nil
	case "!=":
		return actual != expected, nil
	case ">":
		return actual > expected, nil
	case "<":
		return actual < expected, nil
	case ">=":
		return actual >= expected, nil
	case "<=":
		return actual <= expected, nil
	default:
		return false, fmt.Errorf("unknown operator: %s", operator)
	}
}

// PQ REST models for querying party quest instance data

type pqStageStateRestModel struct {
	CustomData map[string]any `json:"customData,omitempty"`
}

type pqInstanceRestModel struct {
	Id         uuid.UUID              `json:"-"`
	StageState pqStageStateRestModel  `json:"stageState"`
}

func (r pqInstanceRestModel) GetName() string {
	return "instances"
}

func (r pqInstanceRestModel) GetID() string {
	return r.Id.String()
}

func (r *pqInstanceRestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid instance ID: %w", err)
	}
	r.Id = id
	return nil
}

func (e *ConditionEvaluator) getPqInstanceByCharacter(characterId uint32) (pqInstanceRestModel, error) {
	url := fmt.Sprintf(requests.RootUrl("PARTY_QUESTS")+"party-quests/instances/character/%d", characterId)
	return requests.GetRequest[pqInstanceRestModel](url)(e.l, e.ctx)
}
