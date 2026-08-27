package asset

import (
	"atlas-channel/cashshop/item"
	"time"

	"github.com/google/uuid"
)

// Model represents a cash shop inventory asset
type Model struct {
	id               uint32
	compartmentId    uuid.UUID
	item             item.Model
	giftFrom         string
	giftMessage      string
	giftAcknowledged bool
	giftNoteSent     bool
}

// Id returns the unique identifier of this asset
func (m Model) Id() uint32 {
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

// CommodityId returns the commodity ID (serial number) of the item
func (m Model) CommodityId() uint32 {
	return m.item.CommodityId()
}

// Quantity returns the quantity of the item
func (m Model) Quantity() uint32 {
	return m.item.Quantity()
}

// Expiration returns the expiration time of the item
func (m Model) Expiration() time.Time {
	return m.item.Expiration()
}

// GiftFrom returns the sender's character name for a gifted asset; empty for
// every other asset.
func (m Model) GiftFrom() string {
	return m.giftFrom
}

// GiftMessage returns the sender's message for a gifted asset; empty for
// every other asset.
func (m Model) GiftMessage() string {
	return m.giftMessage
}

// GiftAcknowledged reports whether the gift list carrying this asset has
// already been presented to the recipient via a LOAD_GIFT_SUCCESS announce
// (task-240 Defect H). This is NOT "the recipient clicked OK" -- see
// atlas-cashshop's asset.Entity.GiftAcknowledged doc comment for the full
// rationale.
func (m Model) GiftAcknowledged() bool {
	return m.giftAcknowledged
}

// GiftNoteSent reports whether the gift-forward note for this asset has
// already been sent to the gifter (task-240 Defect I). This is a SECOND,
// independent flag from GiftAcknowledged -- see atlas-cashshop's
// asset.Entity.GiftNoteSent doc comment for the full rationale.
func (m Model) GiftNoteSent() bool {
	return m.giftNoteSent
}
