package outcome

import (
	"github.com/Chronicle20/atlas/libs/atlas-script-core/condition"
)

// Builder is a builder for Model
type Builder struct {
	conditions []condition.Model
	nextState  string
}

// NewBuilder creates a new Builder
func NewBuilder() *Builder {
	return &Builder{
		conditions: make([]condition.Model, 0),
	}
}

// AddCondition adds an outcome condition
func (b *Builder) AddCondition(cond condition.Model) *Builder {
	b.conditions = append(b.conditions, cond)
	return b
}

// AddConditionFromInput adds an outcome condition from input parameters
func (b *Builder) AddConditionFromInput(condType string, operator string, value string) *Builder {
	cond, err := condition.NewBuilder().
		SetType(condType).
		SetOperator(operator).
		SetValue(value).
		Build()

	if err == nil {
		b.conditions = append(b.conditions, cond)
	}

	return b
}

// SetNextState sets the next state ID
func (b *Builder) SetNextState(nextState string) *Builder {
	b.nextState = nextState
	return b
}

// Build builds the Model
// An empty nextState indicates a terminal state (end of conversation/script)
func (b *Builder) Build() (Model, error) {
	return Model{
		conditions: b.conditions,
		nextState:  b.nextState,
	}, nil
}
