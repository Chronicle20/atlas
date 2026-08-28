package outcome

import (
	"github.com/Chronicle20/atlas/libs/atlas-script-core/condition"
)

// Model represents an outcome with conditions and a next state
type Model struct {
	conditions []condition.Model
	nextState  string
}

// Conditions returns the outcome conditions
func (o Model) Conditions() []condition.Model {
	return o.conditions
}

// NextState returns the next state ID
func (o Model) NextState() string {
	return o.nextState
}
