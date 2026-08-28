package validation

import (
	"fmt"
)

// ConditionBuilder is used to safely construct Condition objects
type ConditionBuilder struct {
	conditionType ConditionType
	operator      Operator
	value         int
	referenceId   *uint32
	step          string
	err           error
}

// NewConditionBuilder creates a new condition builder
func NewConditionBuilder() *ConditionBuilder {
	return &ConditionBuilder{}
}

// SetType sets the condition type
func (b *ConditionBuilder) SetType(condType string) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	switch ConditionType(condType) {
	case JobCondition, MesoCondition, MapCondition, FameCondition, ItemCondition, BuddyCapacityCondition, QuestStatusCondition:
		b.conditionType = ConditionType(condType)
	default:
		b.err = fmt.Errorf("unsupported condition type: %s", condType)
	}
	return b
}

// SetOperator sets the operator
func (b *ConditionBuilder) SetOperator(op string) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	switch Operator(op) {
	case Equals, GreaterThan, LessThan, GreaterEqual, LessEqual:
		b.operator = Operator(op)
	default:
		b.err = fmt.Errorf("unsupported operator: %s", op)
	}
	return b
}

// SetValue sets the value
func (b *ConditionBuilder) SetValue(value int) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	b.value = value
	return b
}

// SetReferenceId sets the reference ID (for quest validation, item conditions, etc.)
func (b *ConditionBuilder) SetReferenceId(referenceId uint32) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	b.referenceId = &referenceId
	return b
}

// SetStep sets the step for quest progress validation
func (b *ConditionBuilder) SetStep(step string) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	b.step = step
	return b
}

// FromInput creates a condition builder from a ConditionInput
func (b *ConditionBuilder) FromInput(input ConditionInput) *ConditionBuilder {
	b.SetType(input.Type)
	b.SetOperator(input.Operator)
	b.SetValue(input.Value)

	if input.ReferenceId != 0 {
		b.SetReferenceId(input.ReferenceId)
	} else if ConditionType(input.Type) == ItemCondition {
		b.err = fmt.Errorf("referenceId is required for item conditions")
	} else if ConditionType(input.Type) == QuestStatusCondition {
		b.err = fmt.Errorf("referenceId is required for quest status conditions")
	}

	if input.Step != "" {
		b.SetStep(input.Step)
	}

	return b
}

// Validate validates the builder state
func (b *ConditionBuilder) Validate() *ConditionBuilder {
	if b.err != nil {
		return b
	}

	// Check if condition type is set
	if b.conditionType == "" {
		b.err = fmt.Errorf("condition type is required")
		return b
	}

	// Check if operator is set
	if b.operator == "" {
		b.err = fmt.Errorf("operator is required")
		return b
	}

	// Check if referenceId is set for item conditions
	if b.conditionType == ItemCondition && b.referenceId == nil {
		b.err = fmt.Errorf("referenceId is required for item conditions")
		return b
	}

	// Check if referenceId is set for quest status conditions
	if b.conditionType == QuestStatusCondition && b.referenceId == nil {
		b.err = fmt.Errorf("referenceId is required for quest status conditions")
		return b
	}

	return b
}

// Build builds a Condition from the builder
func (b *ConditionBuilder) Build() (Condition, error) {
	b.Validate()

	if b.err != nil {
		return Condition{}, b.err
	}

	condition := Condition{
		conditionType: b.conditionType,
		operator:      b.operator,
		value:         b.value,
		step:          b.step,
	}

	if b.referenceId != nil {
		condition.referenceId = *b.referenceId
	}

	return condition, nil
}
