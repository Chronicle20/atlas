package asset

import (
	"time"

	"github.com/google/uuid"
)

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_ASSET_STATUS"
	StatusEventTypeMoved   = "MOVED"
	StatusEventTypeDeleted = "DELETED"
)

// StatusEvent represents an asset status change event from atlas-inventory
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetId       uint32    `json:"assetId"`
	TemplateId    uint32    `json:"templateId"`
	Slot          int16     `json:"slot"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// MovedStatusEventBody contains the previous slot for moved assets
type MovedStatusEventBody struct {
	OldSlot   int16     `json:"oldSlot"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

// DeletedStatusEventBody is empty for deleted assets
type DeletedStatusEventBody struct {
}

// IsEquipmentSlot returns true if the slot is an equipment slot (negative)
func IsEquipmentSlot(slot int16) bool {
	return slot < 0
}

// IsInventorySlot returns true if the slot is an inventory slot (positive)
func IsInventorySlot(slot int16) bool {
	return slot > 0
}

// IsEquipAction returns true if this MOVED event represents equipping an item
// (moved from inventory slot to equipment slot)
func IsEquipAction(oldSlot, newSlot int16) bool {
	return IsInventorySlot(oldSlot) && IsEquipmentSlot(newSlot)
}

// IsUnequipAction returns true if this MOVED event represents unequipping an item
// (moved from equipment slot to inventory slot)
func IsUnequipAction(oldSlot, newSlot int16) bool {
	return IsEquipmentSlot(oldSlot) && IsInventorySlot(newSlot)
}

