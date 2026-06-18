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

// Command is the generic high-level MTS command envelope. TransactionId keys the
// originating saga step (when present); Type discriminates which body is carried.
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// CancelListingCommandBody identifies the listing the seller is cancelling.
type CancelListingCommandBody struct {
	ListingId uuid.UUID `json:"listingId"`
	WorldId   byte      `json:"worldId"`
}

// BuyCommandBody identifies the listing being bought and carries the buyer's
// identity (id + account) plus the seller's account.
type BuyCommandBody struct {
	ListingId       uuid.UUID `json:"listingId"`
	WorldId         byte      `json:"worldId"`
	BuyerId         uint32    `json:"buyerId"`
	BuyerAccountId  uint32    `json:"buyerAccountId"`
	SellerAccountId uint32    `json:"sellerAccountId"`
}

// PlaceBidCommandBody identifies the auction listing being bid on and carries the
// bidder's identity (id + account) plus the raw bid amount.
type PlaceBidCommandBody struct {
	ListingId       uuid.UUID `json:"listingId"`
	WorldId         byte      `json:"worldId"`
	BidderId        uint32    `json:"bidderId"`
	BidderAccountId uint32    `json:"bidderAccountId"`
	Amount          uint32    `json:"amount"`
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

// CreateListingCommandBody initiates a listing. The item snapshot and price
// terms are resolved by atlas-mts from the seller's holding/inventory transfer
// saga; the channel supplies the seller identity, the item being listed, and the
// price terms collected from the register-sale / register-auction packet.
type CreateListingCommandBody struct {
	WorldId         byte   `json:"worldId"`
	SellerId        uint32 `json:"sellerId"`
	SellerAccountId uint32 `json:"sellerAccountId"`
	ItemId          uint32 `json:"itemId"`
	Quantity        uint32 `json:"quantity"`
	Price           uint32 `json:"price"`
	IsAuction       bool   `json:"isAuction"`
	BuyNowPrice     uint32 `json:"buyNowPrice"`
	DurationHours   uint32 `json:"durationHours"`
}

// TakeHomeCommandBody identifies the holding the character is taking home into
// inventory.
type TakeHomeCommandBody struct {
	HoldingId   uuid.UUID `json:"holdingId"`
	WorldId     byte      `json:"worldId"`
	CharacterId uint32    `json:"characterId"`
}
