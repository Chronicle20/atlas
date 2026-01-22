package script

import (
	"fmt"

	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/sirupsen/logrus"
)

// ConditionEvaluator evaluates conditions for reactor scripts
type ConditionEvaluator struct {
	l logrus.FieldLogger
}

// NewConditionEvaluator creates a new condition evaluator
func NewConditionEvaluator(l logrus.FieldLogger) *ConditionEvaluator {
	return &ConditionEvaluator{
		l: l,
	}
}

// EvaluateCondition evaluates a single condition
func (e *ConditionEvaluator) EvaluateCondition(reactorState int8, cond condition.Model) (bool, error) {
	e.l.Debugf("Evaluating condition [%s] with operator [%s] value [%s]", cond.Type(), cond.Operator(), cond.Value())

	switch cond.Type() {
	case "reactor_state":
		return EvaluateReactorStateCondition(reactorState, cond.Operator(), cond.Value())
	default:
		return false, fmt.Errorf("unknown condition type: %s", cond.Type())
	}
}

// EvaluateRule evaluates all conditions for a rule (AND logic)
func (e *ConditionEvaluator) EvaluateRule(reactorState int8, rule Rule) (bool, error) {
	conditions := rule.Conditions()

	// Empty conditions = always match (default rule)
	if len(conditions) == 0 {
		return true, nil
	}

	// All conditions must pass (AND logic)
	for _, cond := range conditions {
		passed, err := e.EvaluateCondition(reactorState, cond)
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
