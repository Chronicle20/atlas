package mts

import (
	"github.com/google/uuid"
)

// FailReason* are SEMANTIC failure keys carried in the BUY_FAILED/BID_FAILED
// events' Reason field. atlas-mts deliberately does NOT speak client wire
// codes: the channel resolves these keys against the tenant writer options
// table "noticeFailReasons" (seed templates; per-version like every other
// dispatcher table) into the client's CITC::NoticeFailReason byte, and falls
// back to the operation's bare *Failed arm when the key or table is absent.
// The seeded tables are IDA-verified against gms v83 (0x5A4752), v84
// (0x5B4C42), v87, and v95. Empty string = no specific reason (bare arm).
const (
	FailReasonGeneric     = ""
	FailReasonNotEnoughNX = "NOT_ENOUGH_NX"
	FailReasonItemSold    = "ITEM_SOLD"
	// FailReasonRegisterFailed is the semantic key for a rejected listing
	// registration (auction duration out of range, price below floor, too many
	// active listings). The v83 client has no registration-specific string, so the
	// tenant noticeFailReasons table maps it to the generic "the request for MTS
	// has failed" notice (CITC::NoticeFailReason default, SP_4808).
	FailReasonRegisterFailed = "REGISTER_FAILED"
)

const (
	// EnvCommandTopic names the high-level MTS command topic. It carries BOTH
	// atlas-mts-local operations (cancel, wish add/remove) AND saga/ticker-driven
	// operations (create, buy, bid, take-home, expire). The local operations are
	// handled directly here; the saga/ticker-driven ones are routed in their own
	// phases.
	EnvCommandTopic = "COMMAND_TOPIC_MTS"

	// --- locally-handled command types (Phase 3) ---

	// CommandCancelListing performs the race-safe active->holding(seller) transition
	// for the seller cancelling their own listing.
	CommandCancelListing = "CANCEL_LISTING"
	// CommandRegisterWish creates a wish-list entry for a character.
	CommandRegisterWish = "REGISTER_WISH"
	// CommandRemoveWish deletes a wish-list entry.
	CommandRemoveWish = "REMOVE_WISH"

	// --- saga/ticker-driven command types (routed in Phases 4-6; declared here so
	// the protocol vocabulary is complete, but intentionally NOT dispatched yet) ---

	// CommandCreateListing creates a listing (saga-driven; routed task-102).
	CommandCreateListing = "CREATE_LISTING"
	// CommandBuy buys out a listing (saga-driven; routed in a later phase).
	CommandBuy = "BUY"
	// CommandPlaceBid places a bid on an auction listing (saga-driven; routed here).
	CommandPlaceBid = "PLACE_BID"
	// CommandTakeHome takes a holding home into inventory (saga-driven; routed task-102).
	CommandTakeHome = "TAKE_HOME"
	// CommandExpireListing expires an auction/listing (ticker-driven; routed in a
	// later phase).
	CommandExpireListing = "EXPIRE_LISTING"
)

// Command is the generic high-level MTS command envelope. TransactionId keys the
// originating saga step (when present); Type discriminates which body is carried.
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// CancelListingCommandBody identifies the listing the seller is cancelling by its
// per-(tenant, world) ITC serial (the client's nITCSN). atlas-mts resolves the
// serial -> listing UUID (listing.GetBySerial), owner-checks the SellerId against
// the listing's seller, then runs the race-safe cancel. The item snapshot is read
// from the listing row by atlas-mts.
type CancelListingCommandBody struct {
	WorldId  byte   `json:"worldId"`
	Serial   uint32 `json:"serial"`
	SellerId uint32 `json:"sellerId"`
}

// BuyCommandBody identifies the listing being bought by its per-(tenant, world)
// ITC serial (the client's nITCSN — the channel never has the listing UUID, only
// the wire serial) and carries the buyer's identity (id + account) from the
// session. atlas-mts resolves the serial -> listing UUID (listing.GetBySerial) and
// reads the seller characterId + account, listValue/buyNowPrice, and commissionRate
// from the listing row (the seller account is captured at list time, so it need not
// be carried). BuyNow distinguishes an immediate-buyout of an auction
// (BUY_AUCTION_IMM, mode 0x14) from a plain fixed-price buy (BUY, mode 0x10).
type BuyCommandBody struct {
	WorldId        byte   `json:"worldId"`
	Serial         uint32 `json:"serial"`
	BuyerId        uint32 `json:"buyerId"`
	BuyerAccountId uint32 `json:"buyerAccountId"`
	BuyNow         bool   `json:"buyNow"`
}

// PlaceBidCommandBody identifies the auction listing being bid on by its
// per-(tenant, world) ITC serial (the client's nITCSN) and carries the bidder's
// identity (id + account) from the session plus the raw bid amount. atlas-mts
// resolves the serial -> listing UUID (listing.GetBySerial) and reads the
// currentBid, minIncrement, listValue, and commissionRate from the listing row.
// The escrow holds the MARKED-UP amount (bid * (1 + commissionRate)); the raw bid
// amount is carried here.
type PlaceBidCommandBody struct {
	WorldId         byte   `json:"worldId"`
	Serial          uint32 `json:"serial"`
	BidderId        uint32 `json:"bidderId"`
	BidderAccountId uint32 `json:"bidderAccountId"`
	Amount          uint32 `json:"amount"`
}

// RegisterWishCommandBody carries the wish-list entry to create. Origin records
// which client ITC arm initiated the wish-add (SET_ZZIM vs REGISTER_WISH_ENTRY)
// so atlas-mts can echo it back on the WISH_ADDED status event; the channel needs
// it to pick the matching clientbound result (SetZzimDone vs RegisterWishItemDone)
// since both arms create the same wish row. See WishOrigin* constants.
type RegisterWishCommandBody struct {
	WishId      uuid.UUID `json:"wishId"`
	WorldId     byte      `json:"worldId"`
	CharacterId uint32    `json:"characterId"`
	ItemId      uint32    `json:"itemId"`
	Price       uint32    `json:"price"`
	Origin      string    `json:"origin"`
}

// RemoveWishCommandBody identifies the wish-list entry to delete. Origin records
// which client ITC arm initiated the wish-remove (DELETE_ZZIM vs CANCEL_WISH) so
// atlas-mts can echo it back on the WISH_REMOVED status event; the channel needs
// it to pick the matching clientbound result (DeleteZzimDone vs
// NotifyCancelWishResult). See WishOrigin* constants.
type RemoveWishCommandBody struct {
	WishId  uuid.UUID `json:"wishId"`
	WorldId byte      `json:"worldId"`
	Origin  string    `json:"origin"`
}

// WishOrigin* discriminate which client ITC_OPERATION arm initiated a wish
// add/remove. They round-trip command -> status event so the channel maps the
// resulting WISH_ADDED/WISH_REMOVED to the correct clientbound result mode.
const (
	WishOriginSetZzim      = "SET_ZZIM"
	WishOriginRegisterWish = "REGISTER_WISH"
	WishOriginDeleteZzim   = "DELETE_ZZIM"
	WishOriginCancelWish   = "CANCEL_WISH"
)

// CreateListingCommandBody initiates a listing (the channel ITC register-sale /
// register-auction / sale-current-item arms emit this). atlas-mts maps it to a
// listing.ListRequest and runs the server-authoritative List flow (price-floor,
// active-cap, auction-duration validation + TransferToMts saga). No serial is
// carried — the listing (and thus its serial) does not exist yet; it is assigned
// when the custody saga's AcceptToMtsListing creates the row.
//
// The seller identity (SellerId/SellerAccountId/SellerName) and the item being
// listed (AssetId = the source inventory slot, SourceInventoryType) come from the
// channel session + the decoded register packet; the item snapshot itself is
// resolved during saga expansion, never trusted from the wire.
type CreateListingCommandBody struct {
	WorldId             byte    `json:"worldId"`
	SellerId            uint32  `json:"sellerId"`
	SellerAccountId     uint32  `json:"sellerAccountId"`
	SellerName          string  `json:"sellerName"`
	SaleType            string  `json:"saleType"`
	SourceInventoryType byte    `json:"sourceInventoryType"`
	AssetId             uint32  `json:"assetId"`
	Quantity            uint32  `json:"quantity"`
	ListValue           uint32  `json:"listValue"`
	BuyNowPrice         *uint32 `json:"buyNowPrice,omitempty"`
	DurationHours       int     `json:"durationHours,omitempty"`
	Category            string  `json:"category"`
	SubCategory         string  `json:"subCategory"`
}

// TakeHomeCommandBody identifies the holding the character is taking home into
// inventory by its per-(tenant, world) ITC serial (the client's nITCSN). atlas-mts
// resolves the serial -> holding UUID (holding.GetBySerial), then runs the
// WithdrawFromMts saga. InventoryType selects the destination tab; Slot is
// advisory (the saga assigns a free slot on expansion).
type TakeHomeCommandBody struct {
	WorldId       byte   `json:"worldId"`
	Serial        uint32 `json:"serial"`
	CharacterId   uint32 `json:"characterId"`
	InventoryType byte   `json:"inventoryType"`
	Slot          int16  `json:"slot"`
}

const (
	// EnvStatusEventTopic names the high-level MTS status/event topic. Every event
	// body carries transactionId + worldId.
	EnvStatusEventTopic = "EVENT_TOPIC_MTS_STATUS"

	// StatusEventTypeListingCreated reports a listing was created.
	StatusEventTypeListingCreated = "LISTING_CREATED"
	// StatusEventTypeListingCancelled reports a listing was cancelled (the item
	// moved to the seller's holding).
	StatusEventTypeListingCancelled = "LISTING_CANCELLED"
	// StatusEventTypeBidPlaced reports a bid was placed on an auction listing.
	StatusEventTypeBidPlaced = "BID_PLACED"
	// StatusEventTypeOutbid reports a prior high bidder was outbid.
	StatusEventTypeOutbid = "OUTBID"
	// StatusEventTypeListingSold reports a listing was sold.
	StatusEventTypeListingSold = "LISTING_SOLD"
	// StatusEventTypeListingExpired reports a listing expired.
	StatusEventTypeListingExpired = "LISTING_EXPIRED"
	// StatusEventTypeItemMovedToHolding reports an item moved into a holding.
	StatusEventTypeItemMovedToHolding = "ITEM_MOVED_TO_HOLDING"
	// StatusEventTypeItemTakenHome reports a holding was taken home into inventory.
	StatusEventTypeItemTakenHome = "ITEM_TAKEN_HOME"
	// StatusEventTypeWishAdded reports a wish-list entry was added.
	StatusEventTypeWishAdded = "WISH_ADDED"
	// StatusEventTypeWishRemoved reports a wish-list entry was removed.
	StatusEventTypeWishRemoved = "WISH_REMOVED"

	// StatusEventTypeListingCreateFailed reports a listing creation was rejected
	// before any custody saga was emitted (price floor, active cap, auction
	// duration, or emit error). The channel writes RegisterSaleEntryFailed to the
	// originating seller.
	StatusEventTypeListingCreateFailed = "LISTING_CREATE_FAILED"
	// StatusEventTypeListingCancelFailed reports a seller's cancel was rejected
	// (serial did not resolve, owner-check failed, or the listing was no longer
	// active — the cancel-vs-buy loser). The channel writes CancelSaleItemFailed to
	// the originating seller.
	StatusEventTypeListingCancelFailed = "LISTING_CANCEL_FAILED"
	// StatusEventTypeBuyFailed reports a buy / buy-now was rejected (serial did not
	// resolve, the listing was not active, or the buyer's prepaid was insufficient).
	// BuyerId is the originating character so the channel can target their session
	// with a BuyItemFailed; Reason is the clientbound NoticeFailReason byte.
	StatusEventTypeBuyFailed = "BUY_FAILED"
	// StatusEventTypeBidFailed reports a place-bid was rejected (serial did not
	// resolve, the listing was not an active auction, the bid was below the floor,
	// or the bid lost the high-bid race). BidderId is the originating character so
	// the channel can target their session with a BidAuctionFailed; Reason is the
	// clientbound NoticeFailReason byte.
	StatusEventTypeBidFailed = "BID_FAILED"
	// StatusEventTypeTakeHomeFailed reports a take-home was rejected (serial did not
	// resolve, owner-check failed, or the withdraw saga could not be emitted). The
	// channel writes MoveItcPurchaseItemLtoSFailed to the originating character.
	StatusEventTypeTakeHomeFailed = "TAKE_HOME_FAILED"
)

// StatusEvent is the generic high-level MTS status/event envelope. TransactionId
// echoes the originating command (when present); Type discriminates the body.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// StatusEventListingCreatedBody reports a created listing.
type StatusEventListingCreatedBody struct {
	WorldId   byte      `json:"worldId"`
	ListingId uuid.UUID `json:"listingId"`
	SellerId  uint32    `json:"sellerId"`
	ItemId    uint32    `json:"itemId"`
}

// StatusEventListingCancelledBody reports a cancelled listing whose item moved to
// the seller's holding.
type StatusEventListingCancelledBody struct {
	WorldId   byte      `json:"worldId"`
	ListingId uuid.UUID `json:"listingId"`
	HoldingId uuid.UUID `json:"holdingId"`
	SellerId  uint32    `json:"sellerId"`
	ItemId    uint32    `json:"itemId"`
}

// StatusEventBidPlacedBody reports a bid placed on an auction.
type StatusEventBidPlacedBody struct {
	WorldId   byte      `json:"worldId"`
	ListingId uuid.UUID `json:"listingId"`
	BidderId  uint32    `json:"bidderId"`
	Amount    uint32    `json:"amount"`
}

// StatusEventOutbidBody reports a prior high bidder was outbid.
type StatusEventOutbidBody struct {
	WorldId          byte      `json:"worldId"`
	ListingId        uuid.UUID `json:"listingId"`
	PreviousBidderId uint32    `json:"previousBidderId"`
}

// StatusEventListingSoldBody reports a sold listing.
type StatusEventListingSoldBody struct {
	WorldId   byte      `json:"worldId"`
	ListingId uuid.UUID `json:"listingId"`
	SellerId  uint32    `json:"sellerId"`
	BuyerId   uint32    `json:"buyerId"`
	ItemId    uint32    `json:"itemId"`
}

// StatusEventListingExpiredBody reports an expired listing.
type StatusEventListingExpiredBody struct {
	WorldId   byte      `json:"worldId"`
	ListingId uuid.UUID `json:"listingId"`
}

// StatusEventItemMovedToHoldingBody reports an item moved into a holding.
type StatusEventItemMovedToHoldingBody struct {
	WorldId   byte      `json:"worldId"`
	HoldingId uuid.UUID `json:"holdingId"`
	OwnerId   uint32    `json:"ownerId"`
	ItemId    uint32    `json:"itemId"`
}

// StatusEventItemTakenHomeBody reports a holding taken home into inventory.
type StatusEventItemTakenHomeBody struct {
	WorldId     byte      `json:"worldId"`
	HoldingId   uuid.UUID `json:"holdingId"`
	CharacterId uint32    `json:"characterId"`
	ItemId      uint32    `json:"itemId"`
}

// StatusEventWishAddedBody reports an added wish-list entry. Origin echoes the
// initiating command's WishOrigin so the channel writes the right clientbound
// result (SetZzimDone vs RegisterWishItemDone).
type StatusEventWishAddedBody struct {
	WorldId     byte      `json:"worldId"`
	WishId      uuid.UUID `json:"wishId"`
	CharacterId uint32    `json:"characterId"`
	ItemId      uint32    `json:"itemId"`
	Origin      string    `json:"origin"`
}

// StatusEventWishRemovedBody reports a removed wish-list entry. Origin echoes the
// initiating command's WishOrigin so the channel writes the right clientbound
// result (DeleteZzimDone vs NotifyCancelWishResult).
type StatusEventWishRemovedBody struct {
	WorldId     byte      `json:"worldId"`
	WishId      uuid.UUID `json:"wishId"`
	CharacterId uint32    `json:"characterId"`
	Origin      string    `json:"origin"`
}

// StatusEventListingCreateFailedBody reports a rejected listing creation. SellerId
// is the originating character so the channel can target the seller's session with
// a RegisterSaleEntryFailed; Reason is the clientbound NoticeFailReason byte.
type StatusEventListingCreateFailedBody struct {
	WorldId  byte   `json:"worldId"`
	SellerId uint32 `json:"sellerId"`
	// ReasonKey is a SEMANTIC failure key the channel resolves through the tenant
	// noticeFailReasons table (like buy/bid). JSON tag "reasonKey" (NOT "reason") so
	// it does not collide with the numeric-reason events on the same topic.
	ReasonKey string `json:"reasonKey,omitempty"`
}

// StatusEventListingCancelFailedBody reports a rejected cancel. SellerId is the
// originating character so the channel can target the seller's session with a
// CancelSaleItemFailed; Reason is the clientbound NoticeFailReason byte.
type StatusEventListingCancelFailedBody struct {
	WorldId  byte   `json:"worldId"`
	Serial   uint32 `json:"serial"`
	SellerId uint32 `json:"sellerId"`
	Reason   byte   `json:"reason"`
}

// StatusEventBuyFailedBody reports a rejected buy / buy-now. BuyerId is the
// originating character so the channel can target their session with a
// BuyItemFailed. ReasonKey is a SEMANTIC failure key the channel resolves through
// the tenant noticeFailReasons table — it uses the JSON tag "reasonKey" (NOT
// "reason") deliberately: EVENT_TOPIC_MTS_STATUS also carries numeric-reason events
// (LISTING_CREATE_FAILED etc.), and since every handler decodes every message, a
// shared "reason" tag with mismatched types (string vs number) makes the other
// handlers fail to unmarshal and drop the message (task-102 live finding).
type StatusEventBuyFailedBody struct {
	WorldId   byte   `json:"worldId"`
	Serial    uint32 `json:"serial"`
	BuyerId   uint32 `json:"buyerId"`
	ReasonKey string `json:"reasonKey,omitempty"`
}

// StatusEventBidFailedBody reports a rejected place-bid. BidderId is the
// originating character so the channel can target their session with a
// BidAuctionFailed. ReasonKey is a semantic failure key (JSON tag "reasonKey", see
// StatusEventBuyFailedBody for why it must not be "reason").
type StatusEventBidFailedBody struct {
	WorldId   byte   `json:"worldId"`
	Serial    uint32 `json:"serial"`
	BidderId  uint32 `json:"bidderId"`
	ReasonKey string `json:"reasonKey,omitempty"`
}

// StatusEventTakeHomeFailedBody reports a rejected take-home. CharacterId is the
// originating character so the channel can target their session with a
// MoveItcPurchaseItemLtoSFailed; Reason is the clientbound NoticeFailReason byte.
type StatusEventTakeHomeFailedBody struct {
	WorldId     byte   `json:"worldId"`
	Serial      uint32 `json:"serial"`
	CharacterId uint32 `json:"characterId"`
	Reason      byte   `json:"reason"`
}
