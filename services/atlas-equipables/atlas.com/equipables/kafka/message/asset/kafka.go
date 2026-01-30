package asset

import (
	"github.com/google/uuid"
	"time"
)

const (
	EnvEventTopicStatus  = "EVENT_TOPIC_ASSET_STATUS"
	StatusEventTypeMoved = "MOVED"
)

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

type MovedStatusEventBody struct {
	OldSlot   int16     `json:"oldSlot"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

// IsEquipEvent returns true if this represents an item being equipped
// (moved from positive inventory slot to negative equipped slot)
func IsEquipEvent(oldSlot, newSlot int16) bool {
	return oldSlot > 0 && newSlot < 0
}

// IsUnequipEvent returns true if this represents an item being unequipped
// (moved from negative equipped slot to positive inventory slot)
func IsUnequipEvent(oldSlot, newSlot int16) bool {
	return oldSlot < 0 && newSlot > 0
}
