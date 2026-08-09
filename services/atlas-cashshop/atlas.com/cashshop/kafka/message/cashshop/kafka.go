package cashshop

import (
	"github.com/google/uuid"

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
	CommandTypeExpire                             = "EXPIRE"
	CommandTypeRequestCouponRedemption            = "REQUEST_COUPON_REDEMPTION"
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

// RequestCouponRedemptionCommandBody carries only the code: the channel has
// already normalized it (trimmed + uppercased), and the owning ACCOUNT is
// resolved service-side from Command.CharacterId, because the packet arrives
// on a character session while wallets are account-scoped.
//
// The v83..v95 clients also send a leading target-character string, but
// targeted redemption (gift coupons) is out of scope (PRD §2) and the field is
// deliberately not carried here.
type RequestCouponRedemptionCommandBody struct {
	Code string `json:"code"`
}

const (
	EnvEventTopicStatus                       = "EVENT_TOPIC_CASH_SHOP_STATUS"
	StatusEventTypeInventoryCapacityIncreased = "INVENTORY_CAPACITY_INCREASED"
	StatusEventTypePurchase                   = "PURCHASE"
	StatusEventTypeError                      = "ERROR"
	StatusEventTypeCouponRedeemed             = "COUPON_REDEEMED"
	StatusEventTypeCouponFailed               = "COUPON_FAILED"
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

// CouponRedeemedBody describes one successful redemption.
//
// MaplePoints and Credit are DELTAS — the amounts this coupon awarded — not
// balances. UseCouponDone.maplePoint is rendered by the client inside a
// "You have received ... using the coupon" sentence and is skipped entirely
// when zero; the balance is refreshed separately by CashQueryResult. See
// docs/tasks/task-206-cash-shop-coupon-codes/derivation.md, "Blocking answer 1".
//
// AssetIds rather than fully-built CashInventoryItem records: the channel
// already owns the asset-id -> CashInventoryItem projection (its purchase
// handler at kafka/consumer/cashshop/consumer.go:105-124), and duplicating it
// here would put packet concerns in atlas-cashshop.
type CouponRedeemedBody struct {
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetIds      []uint32  `json:"assetIds"`
	MaplePoints   uint32    `json:"maplePoints"`
	Credit        uint32    `json:"credit"`
}

// CouponFailedBody carries one of the coupon.ErrorKey* strings.
//
// This is a DISTINCT event type rather than a reuse of StatusEventTypeError,
// because the existing ERROR handler announces
// CashShopInventoryCapacityIncreaseFailedBody — a different mode byte. A
// coupon failure must go out on the USE_COUPON_FAILED arm, so folding it into
// ERROR would force the channel to guess which failure arm an error belongs to.
type CouponFailedBody struct {
	Error string `json:"error"`
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
