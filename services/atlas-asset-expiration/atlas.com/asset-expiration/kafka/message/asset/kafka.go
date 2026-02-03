package asset

import "github.com/google/uuid"

// Topic constants for each service
const (
	EnvCommandTopicStorage     = "COMMAND_TOPIC_STORAGE"
	EnvCommandTopicCashShop    = "COMMAND_TOPIC_CASH_SHOP"
	EnvCommandTopicCompartment = "COMMAND_TOPIC_COMPARTMENT"
)

// Command type constant
const (
	CommandTypeExpire = "EXPIRE"
)

// StorageExpireCommand is sent to atlas-storage to expire an item
type StorageExpireCommand struct {
	TransactionId uuid.UUID          `json:"transactionId"`
	WorldId       byte               `json:"worldId"`
	AccountId     uint32             `json:"accountId"`
	Type          string             `json:"type"`
	Body          StorageExpireBody  `json:"body"`
}

// StorageExpireBody contains the data for expiring a storage item
type StorageExpireBody struct {
	CharacterId    uint32 `json:"characterId"`
	AssetId        uint32 `json:"assetId"`
	TemplateId     uint32 `json:"templateId"`
	InventoryType  int8   `json:"inventoryType"`
	Slot           int16  `json:"slot"`
	ReplaceItemId  uint32 `json:"replaceItemId"`
	ReplaceMessage string `json:"replaceMessage"`
}

// CashShopExpireCommand is sent to atlas-cashshop to expire an item
type CashShopExpireCommand struct {
	CharacterId uint32             `json:"characterId"`
	Type        string             `json:"type"`
	Body        CashShopExpireBody `json:"body"`
}

// CashShopExpireBody contains the data for expiring a cash shop item
type CashShopExpireBody struct {
	AccountId      uint32 `json:"accountId"`
	WorldId        byte   `json:"worldId"`
	AssetId        uint32 `json:"assetId"`
	TemplateId     uint32 `json:"templateId"`
	InventoryType  int8   `json:"inventoryType"`
	Slot           int16  `json:"slot"`
	ReplaceItemId  uint32 `json:"replaceItemId"`
	ReplaceMessage string `json:"replaceMessage"`
}

// CompartmentExpireCommand is sent to atlas-inventory to expire an item
type CompartmentExpireCommand struct {
	TransactionId uuid.UUID            `json:"transactionId"`
	CharacterId   uint32               `json:"characterId"`
	InventoryType byte                 `json:"inventoryType"`
	Type          string               `json:"type"`
	Body          CompartmentExpireBody `json:"body"`
}

// CompartmentExpireBody contains the data for expiring an inventory item
type CompartmentExpireBody struct {
	AssetId        uint32 `json:"assetId"`
	TemplateId     uint32 `json:"templateId"`
	Slot           int16  `json:"slot"`
	ReplaceItemId  uint32 `json:"replaceItemId"`
	ReplaceMessage string `json:"replaceMessage"`
}
