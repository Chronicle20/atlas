package character

import (
	"atlas-character-factory/tenant"
)

const (
	EnvEventTopicCharacterStatus    = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeCreated = "CREATED"

	EnvEventInventoryChanged = "EVENT_TOPIC_INVENTORY_CHANGED"
)

type statusEvent[E any] struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Type        string       `json:"type"`
	WorldId     byte         `json:"worldId"`
	Body        E            `json:"body"`
}

type statusEventCreatedBody struct {
	Name string `json:"name"`
}

type inventoryChangedEvent[M any] struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Slot        int16        `json:"slot"`
	Type        string       `json:"type"`
	Body        M            `json:"body"`
	Silent      bool         `json:"silent"`
}

type inventoryChangedItemAddBody struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

type inventoryChangedItemUpdateBody struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

type inventoryChangedItemMoveBody struct {
	ItemId  uint32 `json:"itemId"`
	OldSlot int16  `json:"oldSlot"`
}
