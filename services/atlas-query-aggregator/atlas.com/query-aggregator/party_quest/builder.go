package party_quest

import (
	"github.com/google/uuid"
)

// Builder provides a builder pattern for creating party quest models
type Builder struct {
	id         uuid.UUID
	customData map[string]string
}

// NewBuilder creates a new party quest model builder
func NewBuilder() *Builder {
	return &Builder{
		customData: make(map[string]string),
	}
}

// SetId sets the instance ID
func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetCustomData sets the custom data map
func (b *Builder) SetCustomData(data map[string]string) *Builder {
	b.customData = data
	return b
}

// Build creates a party quest model from the builder
func (b *Builder) Build() Model {
	return Model{
		id:         b.id,
		customData: b.customData,
	}
}
