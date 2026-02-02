package asset

import (
	"github.com/google/uuid"
	"time"
)

const (
	EnvEventTopicStatus            = "EVENT_TOPIC_ASSET_STATUS"
	StatusEventTypeCreated         = "CREATED"
	StatusEventTypeDeleted         = "DELETED"
	StatusEventTypeMoved           = "MOVED"
	StatusEventTypeQuantityChanged = "QUANTITY_CHANGED"
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

// CreatedStatusEventBody contains info for newly created assets
type CreatedStatusEventBody struct {
	ReferenceId   uint32                 `json:"referenceId"`
	ReferenceType string                 `json:"referenceType"`
	ReferenceData map[string]interface{} `json:"referenceData"`
	Expiration    time.Time              `json:"expiration"`
}

// GetCreatedAt extracts the createdAt timestamp from the ReferenceData if present
func (b CreatedStatusEventBody) GetCreatedAt() time.Time {
	if b.ReferenceData == nil {
		return time.Time{}
	}
	if createdAtStr, ok := b.ReferenceData["createdAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			return t
		}
	}
	return time.Time{}
}

// DeletedStatusEventBody is empty for deleted assets
type DeletedStatusEventBody struct {
}

// MovedStatusEventBody contains the previous slot for moved assets
type MovedStatusEventBody struct {
	OldSlot   int16     `json:"oldSlot"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

// QuantityChangedEventBody contains the new quantity
type QuantityChangedEventBody struct {
	Quantity uint32 `json:"quantity"`
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
