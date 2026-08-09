package cashshop

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic                               = "COMMAND_TOPIC_CASH_SHOP"
	CommandTypeRequestPurchase                    = "REQUEST_PURCHASE"
	CommandTypeRequestInventoryIncreaseByType     = "REQUEST_INVENTORY_INCREASE_BY_TYPE"
	CommandTypeRequestInventoryIncreaseByItem     = "REQUEST_INVENTORY_INCREASE_BY_ITEM"
	CommandTypeRequestStorageIncrease             = "REQUEST_STORAGE_INCREASE"
	CommandTypeRequestStorageIncreaseByItem       = "REQUEST_STORAGE_INCREASE_BY_ITEM"
	CommandTypeRequestCharacterSlotIncreaseByItem = "REQUEST_CHARACTER_SLOT_INCREASE_BY_ITEM"
	CommandTypeMoveFromCashInventory              = "MOVE_FROM_CASH_INVENTORY"
	CommandTypeOpenSurprise                       = "OPEN_SURPRISE"
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

type MoveFromCashInventoryCommandBody struct {
	SerialNumber  uint64 `json:"serialNumber"`
	InventoryType byte   `json:"inventoryType"`
	Slot          int16  `json:"slot"`
}

// OpenSurpriseCommandBody opens one Cash Shop Surprise box. TransactionId is
// minted by atlas-channel per click and is the idempotency key: a Kafka
// redelivery replays the same id (and is rejected by the openings ledger)
// while a genuine second click gets a new one. CashId identifies the box in
// the account's cash locker — the server resolves and re-validates it, since
// the edge does not own the locker.
type OpenSurpriseCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AccountId     uint32    `json:"accountId"`
	CashId        int64     `json:"cashId"`
}

const (
	EnvEventTopicStatus                       = "EVENT_TOPIC_CASH_SHOP_STATUS"
	EventCashShopStatusTypeCharacterEnter     = "CHARACTER_ENTER"
	EventCashShopStatusTypeCharacterExit      = "CHARACTER_EXIT"
	StatusEventTypeInventoryCapacityIncreased = "INVENTORY_CAPACITY_INCREASED"
	StatusEventTypePurchase                   = "PURCHASE"
	StatusEventTypeError                      = "ERROR"
	StatusEventTypeCashItemMovedToInventory   = "CASH_ITEM_MOVED_TO_INVENTORY"
	StatusEventTypeSurpriseOpened             = "SURPRISE_OPENED"
	StatusEventTypeSurpriseFailed             = "SURPRISE_FAILED"
)

// TODO multiple services have different impl of this
type StatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
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

type CharacterMovementBody struct {
	CharacterId uint32     `json:"characterId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
}

type PurchaseEventBody struct {
	TemplateId    uint32    `json:"templateId"`
	Price         uint32    `json:"price"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetId       uint32    `json:"assetId"`
	ItemId        uint32    `json:"itemId"`
}

type CashItemMovedToInventoryEventBody struct {
	CompartmentId uuid.UUID `json:"compartmentId"`
	Slot          int16     `json:"slot"`
}

// SurpriseOpenedEventBody carries everything the channel writer needs for
// the CCashShop::OnCashItemGachaponResult SUCCESS arm. BoxRemaining is the
// box's quantity AFTER the decrement — the client removes the locker row
// when it is 0. RewardCount comes from the commodity catalog, not from the
// pool entry.
type SurpriseOpenedEventBody struct {
	CompartmentId    uuid.UUID `json:"compartmentId"`
	BoxCashId        int64     `json:"boxCashId"`
	BoxRemaining     uint32    `json:"boxRemaining"`
	RewardAssetId    uint32    `json:"rewardAssetId"`
	RewardTemplateId uint32    `json:"rewardTemplateId"`
	RewardCount      uint32    `json:"rewardCount"`
}

// SurpriseFailedEventBody's Reason NEVER reaches the client: the FAILED arm
// of this packet has an empty body and no error-code field (design.md §2.3).
// It exists for the log and for operators. Closed set: BOX_NOT_FOUND,
// NOT_OWNED, NOT_A_SURPRISE_BOX, LOCKER_FULL, POOL_EMPTY, POOL_MISSING,
// COMMODITY_MISSING, INTERNAL.
type SurpriseFailedEventBody struct {
	Reason string `json:"reason"`
}
