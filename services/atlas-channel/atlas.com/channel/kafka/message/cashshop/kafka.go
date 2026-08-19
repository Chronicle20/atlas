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
	CommandTypeRequestCouponRedemption            = "REQUEST_COUPON_REDEMPTION"
	CommandTypeRequestLockerRebate                = "REQUEST_LOCKER_REBATE"
	CommandTypeRequestGiftPurchase                = "REQUEST_GIFT_PURCHASE"
)

type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// RequestPurchaseCommandBody requests one Commodity purchase. TransactionId is
// an opaque correlation id minted by the caller (zero UUID means "no
// correlation" for backward compatibility with existing callers) — see
// OpenSurpriseCommandBody for the same pattern. It is echoed back on both
// PurchaseEventBody and ErrorEventBody so a caller juggling multiple
// concurrent purchases for the same character can tell them apart.
// Operation names the cash shop arm requesting this purchase, so
// atlas-cashshop can echo it back onto PurchaseEventBody / ErrorEventBody
// and the channel can answer on that arm's own SUCCESS/FAILED mode byte
// instead of the generic purchase-success fallback. Empty means "the
// generic BUY arm" -- every producer that predates this field leaves it
// empty and keeps its existing behavior byte for byte.
type RequestPurchaseCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Currency      uint32    `json:"currency"`
	SerialNumber  uint32    `json:"serialNumber"`
	Operation     string    `json:"operation,omitempty"`
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

// RequestLockerRebateCommandBody refunds one locker asset. CashId is the
// client's GW_ItemSlotBase::liCashItemSN (cash_assets.CashId), NOT the row id
// -- see shop_operation_rebate_locker_item.go:18-21. TransactionId is the
// idempotency key, mirroring OpenSurpriseCommandBody: a Kafka redelivery
// replays the same id and is rejected as success-without-effect by the
// ledger, while a genuine second click gets a new one.
type RequestLockerRebateCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AccountId     uint32    `json:"accountId"`
	CashId        int64     `json:"cashId"`
}

// RequestGiftPurchaseCommandBody requests one GIFT purchase (task-240 task
// 13): the sender's wallet is charged and the commodity is delivered into
// the RECIPIENT's locker. The channel resolves the recipient NAME to a
// character id before sending, because atlas-cashshop's character client has
// only GetById -- there is no name lookup there. TransactionId is the
// idempotency key, mirroring RequestLockerRebateCommandBody: a Kafka
// redelivery replays the same id and is rejected as success-without-effect
// by the ledger, while a genuine second click gets a new one.
type RequestGiftPurchaseCommandBody struct {
	TransactionId        uuid.UUID `json:"transactionId"`
	SerialNumber         uint32    `json:"serialNumber"`
	RecipientCharacterId uint32    `json:"recipientCharacterId"`
	SenderName           string    `json:"senderName"`
	Message              string    `json:"message"`
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
	StatusEventTypeCouponRedeemed             = "COUPON_REDEEMED"
	StatusEventTypeCouponFailed               = "COUPON_FAILED"
	StatusEventTypeLockerRebated              = "LOCKER_REBATED"
	StatusEventTypeGiftPurchased              = "GIFT_PURCHASED"
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
	Error         string    `json:"error"`
	CashItemId    uint32    `json:"cashItemId,omitempty"`
	TransactionId uuid.UUID `json:"transactionId"`
	// Operation names the cash shop arm this failure belongs to, so the channel
	// can answer on that arm's own *_FAILED mode byte. Empty means "the legacy
	// capacity-increase arm" -- every producer that predates this field leaves it
	// empty and keeps its existing behavior byte for byte.
	Operation string `json:"operation,omitempty"`
}

const (
	ErrorOperationGift            = "GIFT"
	ErrorOperationBuyNormal       = "BUY_NORMAL"
	ErrorOperationRebate          = "REBATE"
	ErrorOperationCouple          = "COUPLE"
	ErrorOperationFriendship      = "FRIENDSHIP"
	ErrorOperationBuyPackage      = "BUY_PACKAGE"
	ErrorOperationGiftPackage     = "GIFT_PACKAGE"
	ErrorOperationEnableEquipSlot = "ENABLE_EQUIP_SLOT"
)

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
	TransactionId uuid.UUID `json:"transactionId"`
	// Operation is the RequestPurchaseCommandBody.Operation this purchase was
	// requested with, echoed back so the channel can answer on that arm's own
	// SUCCESS mode byte. Empty means the generic BUY arm (today's behavior).
	Operation string `json:"operation,omitempty"`
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

// LockerRebatedBody describes one successful REBATE. Currency is the wallet
// bucket credited (wallet.Model.Balance's convention: 1 = credit/NX, 2 =
// Maple Points, anything else = prepaid) -- see asset.Entity.Currency's doc
// comment for how 0 on the refunded asset resolves to the ordinary credit/NX
// bucket rather than being echoed here as a literal 0. Mirrored for wire
// compatibility with atlas-cashshop; the client-facing announce ignores it
// (CashShopRebateDoneBody carries only sn/amount).
type LockerRebatedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CashId        int64     `json:"cashId"`
	Amount        int32     `json:"amount"`
	Currency      uint32    `json:"currency"`
}

// GiftPurchasedBody describes one successful GIFT. RecipientName is
// resolved server-side (the command only carries RecipientCharacterId) so
// the channel can render the recipient's name without a second round trip.
// There is deliberately no Currency field: a gift is always charged to the
// sender's credit/NX bucket, so there is nothing to echo back that the
// caller does not already know (task-240 task 13/C4).
type GiftPurchasedBody struct {
	TransactionId        uuid.UUID `json:"transactionId"`
	RecipientName        string    `json:"recipientName"`
	TemplateId           uint32    `json:"templateId"`
	Quantity             uint16    `json:"quantity"`
	Price                uint32    `json:"price"`
	RecipientCharacterId uint32    `json:"recipientCharacterId"`
}
