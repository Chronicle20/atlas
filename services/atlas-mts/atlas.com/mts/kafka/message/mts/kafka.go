package mts

import (
	"github.com/google/uuid"
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

	// CommandCreateListing creates a listing (saga-driven; routed in a later phase).
	CommandCreateListing = "CREATE_LISTING"
	// CommandBuy buys out a listing (saga-driven; routed in a later phase).
	CommandBuy = "BUY"
	// CommandPlaceBid places a bid on an auction listing (saga-driven; routed in a
	// later phase).
	CommandPlaceBid = "PLACE_BID"
	// CommandTakeHome takes a holding home into inventory (saga-driven; routed in a
	// later phase).
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

// CancelListingCommandBody identifies the listing the seller is cancelling. The
// seller identity and item snapshot are read from the listing row by atlas-mts.
type CancelListingCommandBody struct {
	ListingId uuid.UUID `json:"listingId"`
	WorldId   byte      `json:"worldId"`
}

// RegisterWishCommandBody carries the wish-list entry to create.
type RegisterWishCommandBody struct {
	WishId      uuid.UUID `json:"wishId"`
	WorldId     byte      `json:"worldId"`
	CharacterId uint32    `json:"characterId"`
	ItemId      uint32    `json:"itemId"`
}

// RemoveWishCommandBody identifies the wish-list entry to delete.
type RemoveWishCommandBody struct {
	WishId  uuid.UUID `json:"wishId"`
	WorldId byte      `json:"worldId"`
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

// StatusEventWishAddedBody reports an added wish-list entry.
type StatusEventWishAddedBody struct {
	WorldId     byte      `json:"worldId"`
	WishId      uuid.UUID `json:"wishId"`
	CharacterId uint32    `json:"characterId"`
	ItemId      uint32    `json:"itemId"`
}

// StatusEventWishRemovedBody reports a removed wish-list entry.
type StatusEventWishRemovedBody struct {
	WorldId     byte      `json:"worldId"`
	WishId      uuid.UUID `json:"wishId"`
	CharacterId uint32    `json:"characterId"`
}
