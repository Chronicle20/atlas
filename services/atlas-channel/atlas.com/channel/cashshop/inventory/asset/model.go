package asset

import (
	"atlas-channel/cashshop/item"
	"time"

	"github.com/google/uuid"
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
