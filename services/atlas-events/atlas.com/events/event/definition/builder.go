package definition

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Builder provides fluent construction of event definition models.
type Builder struct {
	id            uuid.UUID
	theType       string
	name          string
	enabled       bool
	configuration json.RawMessage
	createdAt     time.Time
	updatedAt     time.Time
}

// NewBuilder creates a new builder with the required parameters.
func NewBuilder(theType string, name string) *Builder {
	return &Builder{
		id:      uuid.New(),
		theType: theType,
		name:    name,
	}
}

// SetId sets the definition id.
func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetName sets the definition name.
func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

// SetEnabled sets whether the definition is enabled.
func (b *Builder) SetEnabled(enabled bool) *Builder {
	b.enabled = enabled
	return b
}

// SetConfiguration sets the opaque configuration payload.
func (b *Builder) SetConfiguration(configuration json.RawMessage) *Builder {
	b.configuration = configuration
	return b
}

// SetCreatedAt sets the created-at timestamp.
func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

// SetUpdatedAt sets the updated-at timestamp.
func (b *Builder) SetUpdatedAt(updatedAt time.Time) *Builder {
	b.updatedAt = updatedAt
	return b
}

// Build validates invariants and constructs the final immutable model.
func (b *Builder) Build() (Model, error) {
	if b.theType == "" {
		return Model{}, errors.New("type is required")
	}
	if b.name == "" {
		return Model{}, errors.New("name is required")
	}
	if b.configuration != nil && !json.Valid(b.configuration) {
		return Model{}, errors.New("configuration must be valid JSON")
	}

	return Model{
		id:            b.id,
		theType:       b.theType,
		name:          b.name,
		enabled:       b.enabled,
		configuration: b.configuration,
		createdAt:     b.createdAt,
		updatedAt:     b.updatedAt,
	}, nil
}

// Builder returns a builder initialized with the current model's values.
func (m Model) Builder() *Builder {
	return &Builder{
		id:            m.id,
		theType:       m.theType,
		name:          m.name,
		enabled:       m.enabled,
		configuration: m.configuration,
		createdAt:     m.createdAt,
		updatedAt:     m.updatedAt,
	}
}
