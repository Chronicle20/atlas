package character

import (
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicCharacterStatus    = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeCreated = "CREATED"

	EnvEventInventoryChanged = "EVENT_TOPIC_INVENTORY_CHANGED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	WorldId       world.Id  `json:"worldId"`
	Body          E         `json:"body"`
}

type StatusEventCreatedBody struct {
	Name string `json:"name"`
}

type InventoryChangedEvent[M any] struct {
	CharacterId uint32 `json:"characterId"`
	Slot        int16  `json:"slot"`
	Type        string `json:"type"`
	Body        M      `json:"body"`
	Silent      bool   `json:"silent"`
}

type InventoryChangedItemAddBody struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

type InventoryChangedItemUpdateBody struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

type InventoryChangedItemMoveBody struct {
	ItemId  uint32 `json:"itemId"`
	OldSlot int16  `json:"oldSlot"`
}
