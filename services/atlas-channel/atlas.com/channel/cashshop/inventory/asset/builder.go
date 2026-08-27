package asset

import (
	"atlas-channel/cashshop/item"
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidId is returned when the id is invalid (zero)
var ErrInvalidId = errors.New("id must not be zero")

// ErrInvalidCompartmentId is returned when the compartmentId is invalid (zero UUID)
var ErrInvalidCompartmentId = errors.New("compartmentId must not be zero UUID")

// modelBuilder is a builder for the Model
type builder struct {
	id            uint32
	compartmentId uuid.UUID
	item          item.Model
}

// NewBuilder creates a new builder with required fields
func NewBuilder(id uint32, compartmentId uuid.UUID, i item.Model) *builder {
	return &builder{
		id:            id,
		compartmentId: compartmentId,
		item:          i,
	}
}

// CloneModel creates a builder from this model
func CloneModel(m Model) *builder {
	return &builder{
		id:            m.id,
		compartmentId: m.compartmentId,
		item:          m.item,
	}
}

// SetId sets the id for this builder
func (b *builder) SetId(id uint32) *builder {
	b.id = id
	return b
}

// SetCompartmentId sets the compartmentId for this builder
func (b *builder) SetCompartmentId(compartmentId uuid.UUID) *builder {
	b.compartmentId = compartmentId
	return b
}

// SetItem sets the item associated with this asset
func (b *builder) SetItem(i item.Model) *builder {
	b.item = i
	return b
}

// Build creates a Model from this builder
func (b *builder) Build() (Model, error) {
	if b.id == 0 {
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
func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
