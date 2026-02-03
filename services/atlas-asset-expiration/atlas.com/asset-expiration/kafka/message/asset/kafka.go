package asset

import "github.com/google/uuid"

const (
	EnvCommandTopicAssetExpire = "COMMAND_TOPIC_ASSET_EXPIRE"
)

// ExpireCommand is sent to atlas-inventory, atlas-storage, or atlas-cashshop to expire an item
type ExpireCommand struct {
	TransactionId  uuid.UUID `json:"transactionId"`
	CharacterId    uint32    `json:"characterId"`
	AccountId      uint32    `json:"accountId"`
	WorldId        byte      `json:"worldId"`
	AssetId        uint32    `json:"assetId"`
	TemplateId     uint32    `json:"templateId"`
	InventoryType  int8      `json:"inventoryType"`
	Slot           int16     `json:"slot"`
	ReplaceItemId  uint32    `json:"replaceItemId"`
	ReplaceMessage string    `json:"replaceMessage"`
	Source         string    `json:"source"` // "INVENTORY", "STORAGE", or "CASHSHOP"
}
