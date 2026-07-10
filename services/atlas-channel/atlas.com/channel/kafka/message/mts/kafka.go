package mts

import (
	"github.com/google/uuid"
)

// This file replicates — channel-side — the COMMAND_TOPIC_MTS wire contract
// owned by atlas-mts (services/atlas-mts/atlas.com/mts/kafka/message/mts/
// kafka.go). atlas-channel CANNOT import atlas-mts, so the envelope + command
// bodies are duplicated here with byte-identical JSON tags, exactly as the
// saga orchestrator replicates other services' command structs. Any drift in
// the atlas-mts consumer's expected shape must be mirrored here.
const (
	// EnvCommandTopic names the high-level MTS command topic. It carries BOTH
	// atlas-mts-local operations (cancel, wish add/remove) AND saga/ticker-driven
	// operations (create, buy, bid, take-home, expire).
	EnvCommandTopic = "COMMAND_TOPIC_MTS"

	// CommandCancelListing performs the race-safe active->holding(seller) transition
	// for the seller cancelling their own listing.
	CommandCancelListing = "CANCEL_LISTING"
	// CommandRegisterWish creates a wish-list entry for a character.
	CommandRegisterWish = "REGISTER_WISH"
	// CommandRemoveWish deletes a wish-list entry.
	CommandRemoveWish = "REMOVE_WISH"
	// CommandCreateListing creates a listing (saga-driven).
	CommandCreateListing = "CREATE_LISTING"
	// CommandBuy buys out a listing (saga-driven).
	CommandBuy = "BUY"
	// CommandPlaceBid places a bid on an auction listing.
	CommandPlaceBid = "PLACE_BID"
	// CommandTakeHome takes a holding home into inventory (saga-driven).
	CommandTakeHome = "TAKE_HOME"
)

// ResultKind* discriminate which client result mode a buy/settle should route to
// on its LISTING_SOLD / BUY_FAILED status event. The channel sets it from the ITC
// arm the buyer used (BUY/BUY_AUCTION_IMM -> item, BUY_ZZIM -> zzim, BUY_WISH ->
// wish); atlas-mts's auction settle sets auction_settle. It round-trips command ->
// status event so handleListingSold/handleBuyFailed pick the matching
// CITC::OnNormalItemResult arm. Must match atlas-mts's ResultKind* byte-for-byte.
const (
	ResultKindItem          = "item"
	ResultKindZzim          = "zzim"
	ResultKindWish          = "wish"
	ResultKindAuctionSettle = "auction_settle"
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
// serial -> listing UUID and owner-checks SellerId.
type CancelListingCommandBody struct {
	WorldId  byte   `json:"worldId"`
	Serial   uint32 `json:"serial"`
	SellerId uint32 `json:"sellerId"`
}

// BuyCommandBody identifies the listing being bought by its per-(tenant, world)
// ITC serial (the client's nITCSN — the only addressing the channel has from the
// wire) and carries the buyer's identity (id + account) from the session. atlas-mts
// resolves the serial -> listing UUID and reads the seller account + price terms
// from the listing row. BuyNow distinguishes an immediate-buyout of an auction
// (BUY_AUCTION_IMM, mode 0x14) from a plain fixed-price buy (BUY, mode 0x10).
type BuyCommandBody struct {
	WorldId        byte   `json:"worldId"`
	Serial         uint32 `json:"serial"`
	BuyerId        uint32 `json:"buyerId"`
	BuyerAccountId uint32 `json:"buyerAccountId"`
	BuyNow         bool   `json:"buyNow"`
	// ResultKind records which ITC buy arm initiated this buy (item / zzim / wish)
	// so it round-trips onto the LISTING_SOLD / BUY_FAILED event and the channel
	// picks the matching client result arm.
	ResultKind string `json:"resultKind"`
}

// PlaceBidCommandBody identifies the auction listing being bid on by its
// per-(tenant, world) ITC serial (the client's nITCSN) and carries the bidder's
// identity (id + account) from the session plus the raw bid amount.
type PlaceBidCommandBody struct {
	WorldId         byte   `json:"worldId"`
	Serial          uint32 `json:"serial"`
	BidderId        uint32 `json:"bidderId"`
	BidderAccountId uint32 `json:"bidderAccountId"`
	Amount          uint32 `json:"amount"`
}

// RegisterWishCommandBody carries the wish-list entry to create. Origin records
// which client ITC arm initiated the wish-add (SET_ZZIM vs REGISTER_WISH_ENTRY);
// atlas-mts echoes it on the WISH_ADDED status event so the channel picks the
// matching clientbound result. See WishOrigin* constants.
type RegisterWishCommandBody struct {
	WishId      uuid.UUID `json:"wishId"`
	WorldId     byte      `json:"worldId"`
	CharacterId uint32    `json:"characterId"`
	ItemId      uint32    `json:"itemId"`
	// ListingSerial is the favorited listing's ITC serial for a SET_ZZIM (cart)
	// command, so the cart tracks that exact listing. 0 for a REGISTER_WISH_ENTRY
	// (wanted) command, which references no listing.
	ListingSerial uint32 `json:"listingSerial"`
	Price         uint32 `json:"price"`
	Count         uint32 `json:"count"`
	Origin        string `json:"origin"`
}

// RemoveWishCommandBody identifies the wish-list entry to delete. Origin records
// which client ITC arm initiated the wish-remove (DELETE_ZZIM vs CANCEL_WISH);
// atlas-mts echoes it on the WISH_REMOVED status event. See WishOrigin* constants.
type RemoveWishCommandBody struct {
	WishId  uuid.UUID `json:"wishId"`
	WorldId byte      `json:"worldId"`
	Origin  string    `json:"origin"`
}

// WishOrigin* discriminate which client ITC_OPERATION arm initiated a wish
// add/remove. They round-trip command -> status event so the channel maps the
// resulting WISH_ADDED/WISH_REMOVED to the correct clientbound result mode. Must
// match atlas-mts's WishOrigin* constants byte-for-byte.
const (
	WishOriginSetZzim      = "SET_ZZIM"
	WishOriginRegisterWish = "REGISTER_WISH"
	WishOriginDeleteZzim   = "DELETE_ZZIM"
	WishOriginCancelWish   = "CANCEL_WISH"
	// WishOriginPurchased tags the server-initiated removal of a cart entry when
	// the buyer purchases the favorited item (from the Cart or the browse). It is
	// NOT a client ITC arm: handleListingSold emits REMOVE_WISH with this origin so
	// the bought item leaves the Cart, and handleWishRemoved writes NO client notice
	// for it (BuyItemDone already confirmed the purchase). atlas-mts echoes the
	// origin string verbatim, so no server-side constant is required.
	WishOriginPurchased = "PURCHASED"
)

// CreateListingCommandBody initiates a listing. The item snapshot and price terms
// are resolved by atlas-mts from the seller's inventory transfer saga; the channel
// supplies the seller identity (from the session), the item being listed (AssetId =
// the source inventory slot, SourceInventoryType), the sale terms and (for
// auctions) the duration collected from the register-sale / register-auction
// packet. No serial — the listing does not exist yet. Mirrors atlas-mts's
// CreateListingCommandBody field-for-field.
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
	MinIncrement        uint32  `json:"minIncrement,omitempty"`
	Category            string  `json:"category"`
	SubCategory         string  `json:"subCategory"`
	OfferWishSerial     uint32  `json:"offerWishSerial,omitempty"`
	OfferWishOwnerId    uint32  `json:"offerWishOwnerId,omitempty"`
}

// TakeHomeCommandBody identifies the holding the character is taking home into
// inventory by its per-(tenant, world) ITC serial (the client's nITCSN). atlas-mts
// resolves the serial -> holding UUID. Mirrors atlas-mts's TakeHomeCommandBody.
type TakeHomeCommandBody struct {
	WorldId       byte   `json:"worldId"`
	Serial        uint32 `json:"serial"`
	CharacterId   uint32 `json:"characterId"`
	InventoryType byte   `json:"inventoryType"`
	Slot          int16  `json:"slot"`
}

// --- EVENT_TOPIC_MTS_STATUS (status events the channel consumes) --------------
//
// These mirror atlas-mts's StatusEvent envelope + bodies (services/atlas-mts/
// atlas.com/mts/kafka/message/mts/kafka.go). The channel consumes them to write
// the matching clientbound MtsOperation* result to the originating character's
// session. Only the event types the channel acts on are mirrored here.
const (
	// EnvStatusEventTopic names the high-level MTS status/event topic.
	EnvStatusEventTopic = "EVENT_TOPIC_MTS_STATUS"

	StatusEventTypeListingCreated      = "LISTING_CREATED"
	StatusEventTypeListingCancelled    = "LISTING_CANCELLED"
	StatusEventTypeItemTakenHome       = "ITEM_TAKEN_HOME"
	StatusEventTypeListingCreateFailed = "LISTING_CREATE_FAILED"
	StatusEventTypeListingCancelFailed = "LISTING_CANCEL_FAILED"
	StatusEventTypeTakeHomeFailed      = "TAKE_HOME_FAILED"
	StatusEventTypeListingSold         = "LISTING_SOLD"
	StatusEventTypeBuyFailed           = "BUY_FAILED"
	StatusEventTypeBidFailed           = "BID_FAILED"
	StatusEventTypeBidPlaced           = "BID_PLACED"
	StatusEventTypeOutbid              = "OUTBID"
	StatusEventTypeWishAdded           = "WISH_ADDED"
	StatusEventTypeWishRemoved         = "WISH_REMOVED"
)

// StatusEvent is the generic high-level MTS status/event envelope.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// StatusEventListingCreatedBody reports a created listing. SellerId is the target
// character for the RegisterSaleEntryDone result. SaleType distinguishes a normal
// listing-creation (fixed/auction) from an offer creation ("offer"), which routes
// to SaleCurrentItemToWishDone instead. Mirrors atlas-mts's field-for-field.
type StatusEventListingCreatedBody struct {
	WorldId   byte      `json:"worldId"`
	ListingId uuid.UUID `json:"listingId"`
	SellerId  uint32    `json:"sellerId"`
	ItemId    uint32    `json:"itemId"`
	SaleType  string    `json:"saleType"`
}

// StatusEventListingCancelledBody reports a cancelled listing. SellerId is the
// target character for the CancelSaleItemDone result.
type StatusEventListingCancelledBody struct {
	WorldId   byte      `json:"worldId"`
	ListingId uuid.UUID `json:"listingId"`
	HoldingId uuid.UUID `json:"holdingId"`
	SellerId  uint32    `json:"sellerId"`
	ItemId    uint32    `json:"itemId"`
}

// StatusEventItemTakenHomeBody reports a holding taken home. CharacterId is the
// target character for the MoveItcPurchaseItemLtoSDone result.
type StatusEventItemTakenHomeBody struct {
	WorldId     byte      `json:"worldId"`
	HoldingId   uuid.UUID `json:"holdingId"`
	CharacterId uint32    `json:"characterId"`
	ItemId      uint32    `json:"itemId"`
}

// StatusEventListingCreateFailedBody reports a rejected listing creation. SellerId
// is the target character for the RegisterSaleEntryFailed result. ReasonKey is a
// semantic failure key resolved via the tenant noticeFailReasons table (JSON tag
// "reasonKey", not "reason" — see StatusEventBuyFailedBody for the collision
// rationale).
type StatusEventListingCreateFailedBody struct {
	WorldId   byte   `json:"worldId"`
	SellerId  uint32 `json:"sellerId"`
	ReasonKey string `json:"reasonKey,omitempty"`
}

// StatusEventListingCancelFailedBody reports a rejected cancel. SellerId is the
// target character for the CancelSaleItemFailed result. ReasonKey is a SEMANTIC
// failure key resolved through the tenant noticeFailReasons table (DOM-25); JSON
// tag "reasonKey" (NOT "reason"), matching buy/bid — see StatusEventBuyFailedBody.
type StatusEventListingCancelFailedBody struct {
	WorldId   byte   `json:"worldId"`
	Serial    uint32 `json:"serial"`
	SellerId  uint32 `json:"sellerId"`
	ReasonKey string `json:"reasonKey,omitempty"`
}

// StatusEventTakeHomeFailedBody reports a rejected take-home. CharacterId is the
// target character for the MoveItcPurchaseItemLtoSFailed result. ReasonKey is a
// SEMANTIC failure key resolved through the tenant noticeFailReasons table
// (DOM-25); JSON tag "reasonKey" (NOT "reason") — see StatusEventBuyFailedBody.
type StatusEventTakeHomeFailedBody struct {
	WorldId     byte   `json:"worldId"`
	Serial      uint32 `json:"serial"`
	CharacterId uint32 `json:"characterId"`
	ReasonKey   string `json:"reasonKey,omitempty"`
}

// StatusEventBidPlacedBody reports a bid placed on an auction. BidderId is the
// character whose prepaid the escrow debit just left — the channel refreshes their
// NX counter.
type StatusEventBidPlacedBody struct {
	WorldId   byte      `json:"worldId"`
	ListingId uuid.UUID `json:"listingId"`
	BidderId  uint32    `json:"bidderId"`
	Amount    uint32    `json:"amount"`
}

// StatusEventOutbidBody reports a prior high bidder was outbid. PreviousBidderId is
// the character whose escrow was released back to prepaid — the channel refreshes
// their NX counter.
type StatusEventOutbidBody struct {
	WorldId          byte      `json:"worldId"`
	ListingId        uuid.UUID `json:"listingId"`
	PreviousBidderId uint32    `json:"previousBidderId"`
}

// StatusEventListingSoldBody reports a sold listing. BuyerId is the target
// character for the BuyItemDone result. SaleType distinguishes a normal buy
// (fixed/auction) from an offer purchase ("offer"), which routes to BuyWishDone
// (want-ad accept) instead. Mirrors atlas-mts's field-for-field.
type StatusEventListingSoldBody struct {
	WorldId   byte      `json:"worldId"`
	ListingId uuid.UUID `json:"listingId"`
	SellerId  uint32    `json:"sellerId"`
	BuyerId   uint32    `json:"buyerId"`
	ItemId    uint32    `json:"itemId"`
	SaleType  string    `json:"saleType"`
	// ResultKind selects which client result arm the channel routes this sold notice
	// to (item -> BuyItemDone, zzim -> BuyZzimItemDone, wish -> BuyWishDone,
	// auction_settle -> SuccessBidInfoResult). Price is the settled BASE price,
	// carried for the auction-settle SuccessBidInfo arm.
	ResultKind string `json:"resultKind"`
	Price      uint32 `json:"price"`
}

// StatusEventBuyFailedBody reports a rejected buy / buy-now. BuyerId is the target
// character for the BuyItemFailed result. ReasonKey is a semantic failure key
// resolved via the tenant noticeFailReasons table; its JSON tag is "reasonKey" (NOT
// "reason") to avoid a decode collision with the numeric-reason events on the same
// topic (see the atlas-mts mirror for the full rationale).
type StatusEventBuyFailedBody struct {
	WorldId   byte   `json:"worldId"`
	Serial    uint32 `json:"serial"`
	BuyerId   uint32 `json:"buyerId"`
	ReasonKey string `json:"reasonKey,omitempty"`
	// ResultKind selects which bare failed arm the channel routes to (item ->
	// BuyItemFailed, zzim -> BuyZzimItemFailed, wish -> BuyWishFailed) before the
	// noticeFailReasons wrapper is applied.
	ResultKind string `json:"resultKind"`
}

// StatusEventBidFailedBody reports a rejected place-bid. BidderId is the target
// character for the BidAuctionFailed result. ReasonKey is a semantic failure key
// (JSON tag "reasonKey"; see StatusEventBuyFailedBody).
type StatusEventBidFailedBody struct {
	WorldId   byte   `json:"worldId"`
	Serial    uint32 `json:"serial"`
	BidderId  uint32 `json:"bidderId"`
	ReasonKey string `json:"reasonKey,omitempty"`
}

// StatusEventWishAddedBody reports an added wish-list entry. CharacterId is the
// target character for the wish-add result; Origin discriminates which ITC arm
// initiated the add (SET_ZZIM -> SetZzimDone, REGISTER_WISH -> RegisterWishItemDone).
type StatusEventWishAddedBody struct {
	WorldId     byte      `json:"worldId"`
	WishId      uuid.UUID `json:"wishId"`
	CharacterId uint32    `json:"characterId"`
	ItemId      uint32    `json:"itemId"`
	Origin      string    `json:"origin"`
}

// StatusEventWishRemovedBody reports a removed wish-list entry. CharacterId is the
// target character for the wish-remove result; Origin discriminates which ITC arm
// initiated the remove (DELETE_ZZIM -> DeleteZzimDone, CANCEL_WISH ->
// NotifyCancelWishResult).
type StatusEventWishRemovedBody struct {
	WorldId     byte      `json:"worldId"`
	WishId      uuid.UUID `json:"wishId"`
	CharacterId uint32    `json:"characterId"`
	Origin      string    `json:"origin"`
}
