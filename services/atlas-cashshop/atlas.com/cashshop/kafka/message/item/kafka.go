package item

import "github.com/google/uuid"

const (
	EnvCommandTopic            = "COMMAND_TOPIC_CASH_ITEM"
	EnvStatusTopic             = "STATUS_TOPIC_CASH_ITEM"
	EnvCommandTopicAssetExpire = "COMMAND_TOPIC_ASSET_EXPIRE"

	CommandCreate = "CREATE"

	StatusCreated = "CREATED"
	StatusExpired = "EXPIRED"
)

type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type CreateCommandBody struct {
	TemplateId  uint32 `json:"templateId"`
	CommodityId uint32 `json:"commodityId"`
	Quantity    uint32 `json:"quantity"`
	PurchasedBy uint32 `json:"purchasedBy"`
}

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StatusEventCreatedBody struct {
	CashId      int64  `json:"cashId"`
	TemplateId  uint32 `json:"templateId"`
	Quantity    uint32 `json:"quantity"`
	PurchasedBy uint32 `json:"purchasedBy"`
	Flag        uint16 `json:"flag"`
}

// ExpireCommand is received from atlas-asset-expiration to expire a cash shop item
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

// StatusEventExpiredBody contains information about an expired item
type StatusEventExpiredBody struct {
	IsCash         bool   `json:"isCash"`
	ReplaceItemId  uint32 `json:"replaceItemId,omitempty"`
	ReplaceMessage string `json:"replaceMessage,omitempty"`
}
