package cashshop

import (
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic                               = "COMMAND_TOPIC_CASH_SHOP"
	CommandTypeRequestPurchase                    = "REQUEST_PURCHASE"
	CommandTypeRequestInventoryIncreaseByType     = "REQUEST_INVENTORY_INCREASE_BY_TYPE"
	CommandTypeRequestInventoryIncreaseByItem     = "REQUEST_INVENTORY_INCREASE_BY_ITEM"
	CommandTypeRequestStorageIncrease             = "REQUEST_STORAGE_INCREASE"
	CommandTypeRequestStorageIncreaseByItem       = "REQUEST_STORAGE_INCREASE_BY_ITEM"
	CommandTypeRequestCharacterSlotIncreaseByItem = "REQUEST_CHARACTER_SLOT_INCREASE_BY_ITEM"
	CommandTypeExpire                             = "EXPIRE"
)

type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type RequestPurchaseCommandBody struct {
	Currency     uint32 `json:"currency"`
	SerialNumber uint32 `json:"serialNumber"`
}

type RequestInventoryIncreaseByTypeCommandBody struct {
	Currency      uint32 `json:"currency"`
	InventoryType byte   `json:"inventoryType"`
}

type RequestInventoryIncreaseByItemCommandBody struct {
	Currency     uint32 `json:"currency"`
	SerialNumber uint32 `json:"serialNumber"`
}

type RequestStorageIncreaseBody struct {
	Currency uint32 `json:"currency"`
}

type RequestStorageIncreaseByItemCommandBody struct {
	Currency     uint32 `json:"currency"`
	SerialNumber uint32 `json:"serialNumber"`
}

type RequestCharacterSlotIncreaseByItemCommandBody struct {
	Currency     uint32 `json:"currency"`
	SerialNumber uint32 `json:"serialNumber"`
}

const (
	EnvEventTopicStatus                       = "EVENT_TOPIC_CASH_SHOP_STATUS"
	StatusEventTypeInventoryCapacityIncreased = "INVENTORY_CAPACITY_INCREASED"
	StatusEventTypePurchase                   = "PURCHASE"
	StatusEventTypeError                      = "ERROR"
)

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type InventoryCapacityIncreasedBody struct {
	InventoryType byte   `json:"inventoryType"`
	Capacity      uint32 `json:"capacity"`
	Amount        uint32 `json:"amount"`
}

type ErrorEventBody struct {
	Error      string `json:"error"`
	CashItemId uint32 `json:"cashItemId,omitempty"`
}

type PurchaseEventBody struct {
	TemplateId    uint32    `json:"templateId"`
	Price         uint32    `json:"price"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetId       uint32    `json:"assetId"`
}

// ExpireCommandBody contains the data for expiring a cash shop item
type ExpireCommandBody struct {
	AccountId      uint32   `json:"accountId"`
	WorldId        world.Id `json:"worldId"`
	AssetId        uint32   `json:"assetId"`
	TemplateId     uint32   `json:"templateId"`
	InventoryType  int8     `json:"inventoryType"`
	Slot           int16    `json:"slot"`
	ReplaceItemId  uint32   `json:"replaceItemId"`
	ReplaceMessage string   `json:"replaceMessage"`
}
