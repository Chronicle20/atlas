package operation

import (
	"errors"
)

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
