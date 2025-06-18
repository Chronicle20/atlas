package asset

import (
	"atlas-cashshop/cashshop/item"
	"github.com/google/uuid"
	"time"
)

// Model represents a cash shop inventory asset
type Model struct {
	id            uuid.UUID
	compartmentId uuid.UUID
	item          item.Model
}

// Id returns the unique identifier of this asset
func (m Model) Id() uuid.UUID {
	return m.id
}

// CompartmentId returns the compartment ID this asset belongs to
func (m Model) CompartmentId() uuid.UUID {
	return m.compartmentId
}

// Item returns the item associated with this asset
func (m Model) Item() item.Model {
	return m.item
}

// TemplateId returns the template ID of the item
func (m Model) TemplateId() uint32 {
	return m.item.TemplateId()
}

// Quantity returns the quantity of the item
func (m Model) Quantity() uint32 {
	return m.item.Quantity()
}

// Expiration returns the expiration time of the item
func (m Model) Expiration() time.Time {
	return m.item.Expiration()
}

// Clone creates a builder from this model
func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:            m.id,
		compartmentId: m.compartmentId,
		item:          m.item,
	}
}

// ModelBuilder is a builder for the Model
type ModelBuilder struct {
	id            uuid.UUID
	compartmentId uuid.UUID
	item          item.Model
}

// NewBuilder creates a new ModelBuilder
func NewBuilder(id uuid.UUID, compartmentId uuid.UUID, item item.Model) *ModelBuilder {
	return &ModelBuilder{
		id:            id,
		compartmentId: compartmentId,
		item:          item,
	}
}

// SetItem sets the item associated with this asset
func (b *ModelBuilder) SetItem(item item.Model) *ModelBuilder {
	b.item = item
	return b
}

// Build creates a Model from this builder
func (b *ModelBuilder) Build() Model {
	return Model{
		id:            b.id,
		compartmentId: b.compartmentId,
		item:          b.item,
	}
}
