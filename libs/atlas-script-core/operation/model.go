package operation

import (
	"errors"

	"github.com/Chronicle20/atlas-constants/field"
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

// Builder is a builder for Model
type Builder struct {
	operationType string
	params        map[string]string
}

// NewBuilder creates a new Builder
func NewBuilder() *Builder {
	return &Builder{
		params: make(map[string]string),
	}
}

// SetType sets the operation type
func (b *Builder) SetType(operationType string) *Builder {
	b.operationType = operationType
	return b
}

// SetParams sets the operation parameters
func (b *Builder) SetParams(params map[string]string) *Builder {
	b.params = params
	return b
}

// AddParamValue adds an operation parameter value
func (b *Builder) AddParamValue(key string, value string) *Builder {
	b.params[key] = value
	return b
}

// Build builds the Model
func (b *Builder) Build() (Model, error) {
	if b.operationType == "" {
		return Model{}, errors.New("type is required")
	}

	return Model{
		operationType: b.operationType,
		params:        b.params,
	}, nil
}

// Executor is the interface for executing operations
type Executor interface {
	// ExecuteOperation executes a single operation for a character
	ExecuteOperation(field field.Model, characterId uint32, operation Model) error

	// ExecuteOperations executes multiple operations for a character
	ExecuteOperations(field field.Model, characterId uint32, operations []Model) error
}
