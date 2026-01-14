package asset

import (
	"atlas-channel/cashshop/item"
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidId is returned when the id is invalid (zero UUID)
var ErrInvalidId = errors.New("id must not be zero UUID")

// ErrInvalidCompartmentId is returned when the compartmentId is invalid (zero UUID)
var ErrInvalidCompartmentId = errors.New("compartmentId must not be zero UUID")

// modelBuilder is a builder for the Model
type modelBuilder struct {
	id            uuid.UUID
	compartmentId uuid.UUID
	item          item.Model
}

// NewModelBuilder creates a new modelBuilder with required fields
func NewModelBuilder(id uuid.UUID, compartmentId uuid.UUID, i item.Model) *modelBuilder {
	return &modelBuilder{
		id:            id,
		compartmentId: compartmentId,
		item:          i,
	}
}

// CloneModel creates a builder from this model
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:            m.id,
		compartmentId: m.compartmentId,
		item:          m.item,
	}
}

// SetId sets the id for this builder
func (b *modelBuilder) SetId(id uuid.UUID) *modelBuilder {
	b.id = id
	return b
}

// SetCompartmentId sets the compartmentId for this builder
func (b *modelBuilder) SetCompartmentId(compartmentId uuid.UUID) *modelBuilder {
	b.compartmentId = compartmentId
	return b
}

// SetItem sets the item associated with this asset
func (b *modelBuilder) SetItem(i item.Model) *modelBuilder {
	b.item = i
	return b
}

// Build creates a Model from this builder
func (b *modelBuilder) Build() (Model, error) {
	if b.id == uuid.Nil {
		return Model{}, ErrInvalidId
	}
	if b.compartmentId == uuid.Nil {
		return Model{}, ErrInvalidCompartmentId
	}
	return Model{
		id:            b.id,
		compartmentId: b.compartmentId,
		item:          b.item,
	}, nil
}

// MustBuild creates a Model from this builder and panics if validation fails
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
