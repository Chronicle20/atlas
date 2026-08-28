package operation

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Model represents an operation to be executed
type Model struct {
	operationType string
	params        map[string]string
}

// Type returns the operation type
func (o Model) Type() string {
	return o.operationType
}

// Params returns the operation parameters
func (o Model) Params() map[string]string {
	return o.params
}

// Executor is the interface for executing operations
type Executor interface {
	// ExecuteOperation executes a single operation for a character
	ExecuteOperation(field field.Model, characterId uint32, operation Model) error

	// ExecuteOperations executes multiple operations for a character
	ExecuteOperations(field field.Model, characterId uint32, operations []Model) error
}
