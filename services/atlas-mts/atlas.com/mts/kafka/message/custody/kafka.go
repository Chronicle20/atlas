package custody

import (
	"time"

	"github.com/google/uuid"
)

const (
	// EnvCommandTopic is the env var naming the MTS custody command topic. The
	// saga orchestrator dispatches AcceptToMtsListing / ReleaseFromMtsHolding
	// commands here, mirroring COMMAND_TOPIC_CASH_COMPARTMENT.
	EnvCommandTopic = "COMMAND_TOPIC_MTS_CUSTODY"

	// CommandAcceptToMtsListing creates the listing row in active state from the
	// carried item snapshot (the item has already left the seller inventory).
	CommandAcceptToMtsListing = "ACCEPT_TO_MTS_LISTING"
	// CommandReleaseFromMtsHolding soft-deletes a take-home holding row.
	CommandReleaseFromMtsHolding = "RELEASE_FROM_MTS_HOLDING"
	// CommandMtsMoveListingToHolding marks a sold listing's row `sold` and creates
	// the buyer's `purchased` holding from the listing's snapshot, in one tx.
	CommandMtsMoveListingToHolding = "MTS_MOVE_LISTING_TO_HOLDING"
	// CommandRestoreMtsHolding un-soft-deletes a holding row by id. It is the
	// inverse of ReleaseFromMtsHolding, dispatched by the saga compensator when a
	// WithdrawFromMts saga fails AFTER the holding was released (the take-home
	// AcceptToCharacter step failed): the released holding must be restored so the
	// item is not lost. Idempotent: clearing deleted_at on an already-live row
	// affects 0 rows and is still success.
	CommandRestoreMtsHolding = "RESTORE_MTS_HOLDING"
	// CommandRemoveMtsListing hard-deletes a spurious ACTIVE listing row by id —
	// the late-compensation inverse of AcceptToMtsListing. When a list saga times
	// out after release_from_character (item left inventory) and its compensation
	// re-grants the item to the seller, a late-successful accept still creates the
	// listing, duplicating the item. Removing the (still-active) listing deletes
	// the duplicate; the delete is guarded to state=active so a listing that was
	// bought/cancelled in the interim is left untouched (0 rows = success).
	CommandRemoveMtsListing = "REMOVE_MTS_LISTING"
	// CommandRestoreListingFromHolding reverses a settlement move — the
	// late-compensation inverse of MtsMoveListingToHolding. When a buy saga times
	// out (the buyer's prepaid debit already compensated) but the move lands late,
	// the item was delivered to the buyer's holding and the listing marked sold.
	// This soft-deletes the deterministic buyer holding (moveHoldingId(listingId,
	// buyerId)) and transitions the listing sold->active, in one tx, so the item
	// returns to the marketplace and the buyer keeps nothing (no free item).
	CommandRestoreListingFromHolding = "RESTORE_LISTING_FROM_HOLDING"
)

// Command is the generic custody command envelope. TransactionId keys the saga
// step; Type discriminates which body is carried.
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// AcceptToMtsListingCommandBody carries every field needed to CREATE a listing
// row in active state. The item is already gone from inventory, so its full
// snapshot (template, quantity, equip stat block) travels in the command, along
// with the sale/auction params. ListingId is allocated up-front by the
// initiator so creation is deterministic and idempotent on replay.
type AcceptToMtsListingCommandBody struct {
	ListingId       uuid.UUID `json:"listingId"`
	WorldId         byte      `json:"worldId"`
	SellerId        uint32    `json:"sellerId"`
	SellerAccountId uint32    `json:"sellerAccountId"`
	SellerName      string    `json:"sellerName"`
	SaleType        string    `json:"saleType"`

	// item snapshot
	TemplateId uint32 `json:"templateId"`
	Quantity   uint32 `json:"quantity"`

	// equip stat block
	Strength      uint16 `json:"strength"`
	Dexterity     uint16 `json:"dexterity"`
	Intelligence  uint16 `json:"intelligence"`
	Luck          uint16 `json:"luck"`
	HP            uint16 `json:"hp"`
	MP            uint16 `json:"mp"`
	WeaponAttack  uint16 `json:"weaponAttack"`
	MagicAttack   uint16 `json:"magicAttack"`
	WeaponDefense uint16 `json:"weaponDefense"`
	MagicDefense  uint16 `json:"magicDefense"`
	Accuracy      uint16 `json:"accuracy"`
	Avoidability  uint16 `json:"avoidability"`
	Hands         uint16 `json:"hands"`
	Speed         uint16 `json:"speed"`
	Jump          uint16 `json:"jump"`
	Slots         uint16 `json:"slots"`
	Level         byte   `json:"level"`
	ItemLevel     byte   `json:"itemLevel"`
	ItemExp       uint32 `json:"itemExp"`
	RingId        uint32 `json:"ringId"`
	ViciousCount  uint32 `json:"viciousCount"`
	Flags         uint16 `json:"flags"`

	// sale params
	ListValue      uint32     `json:"listValue"`
	BuyNowPrice    *uint32    `json:"buyNowPrice"`
	CommissionRate float64    `json:"commissionRate"`
	Category       string     `json:"category"`
	SubCategory    string     `json:"subCategory"`
	EndsAt         *time.Time `json:"endsAt"`
	MinIncrement   uint32     `json:"minIncrement"`

	// offer link: the want-ad this `offer` listing fulfills (0 for non-offers).
	OfferWishSerial  uint32 `json:"offerWishSerial"`
	OfferWishOwnerId uint32 `json:"offerWishOwnerId"`
}

// ReleaseFromMtsHoldingCommandBody soft-deletes the take-home holding row.
type ReleaseFromMtsHoldingCommandBody struct {
	HoldingId uuid.UUID `json:"holdingId"`
}

// RestoreMtsHoldingCommandBody un-soft-deletes the holding row by id (the
// compensating inverse of ReleaseFromMtsHolding).
type RestoreMtsHoldingCommandBody struct {
	HoldingId uuid.UUID `json:"holdingId"`
}

// MtsMoveListingToHoldingCommandBody carries the listing to settle plus the
// buyer/world identity for the holding to create. The item snapshot is read from
// the listing row by atlas-mts (not carried here), since the listing already
// holds it.
type MtsMoveListingToHoldingCommandBody struct {
	ListingId uuid.UUID `json:"listingId"`
	BuyerId   uint32    `json:"buyerId"`
	WorldId   byte      `json:"worldId"`
}

// RemoveMtsListingCommandBody hard-deletes a spurious active listing by id (the
// late-compensation inverse of AcceptToMtsListing).
type RemoveMtsListingCommandBody struct {
	ListingId uuid.UUID `json:"listingId"`
}

// RestoreListingFromHoldingCommandBody reverses a settlement move: it identifies
// the buyer holding to soft-delete via (listingId, buyerId) — the same
// deterministic pair the forward move derived the holding id from — and the
// listing to transition sold->active (the inverse of MtsMoveListingToHolding).
type RestoreListingFromHoldingCommandBody struct {
	ListingId uuid.UUID `json:"listingId"`
	BuyerId   uint32    `json:"buyerId"`
}

const (
	// EnvStatusEventTopic names the custody status (ack) topic.
	EnvStatusEventTopic = "EVENT_TOPIC_MTS_CUSTODY_STATUS"

	// StatusEventTypeAccepted acks an AcceptToMtsListing command (row created or
	// already present — both are success).
	StatusEventTypeAccepted = "ACCEPTED"
	// StatusEventTypeReleased acks a ReleaseFromMtsHolding command (row
	// soft-deleted or already released — both are success).
	StatusEventTypeReleased = "RELEASED"
	// StatusEventTypeMoved acks an MtsMoveListingToHolding command (listing marked
	// sold and buyer holding created, or already moved on replay — both success).
	StatusEventTypeMoved = "MOVED"
	// StatusEventTypeRestored acks a RestoreMtsHolding command (row un-soft-deleted
	// or already live — both success).
	StatusEventTypeRestored = "RESTORED"
	// StatusEventTypeError reports a custody failure.
	StatusEventTypeError = "ERROR"
)

// StatusEvent is the generic custody ack envelope. TransactionId echoes the
// command so the orchestrator can complete/fail the saga step.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// StatusEventAcceptedBody acks listing creation, echoing the listing id.
type StatusEventAcceptedBody struct {
	ListingId uuid.UUID `json:"listingId"`
}

// StatusEventReleasedBody acks a holding release, echoing the holding id.
type StatusEventReleasedBody struct {
	HoldingId uuid.UUID `json:"holdingId"`
}

// StatusEventRestoredBody acks a holding restore, echoing the holding id.
type StatusEventRestoredBody struct {
	HoldingId uuid.UUID `json:"holdingId"`
}

// StatusEventMovedBody acks a settlement move, echoing the listing id and the
// created buyer holding id.
type StatusEventMovedBody struct {
	ListingId uuid.UUID `json:"listingId"`
	HoldingId uuid.UUID `json:"holdingId"`
}

// StatusEventErrorBody reports a custody error.
type StatusEventErrorBody struct {
	Error string `json:"error"`
}
