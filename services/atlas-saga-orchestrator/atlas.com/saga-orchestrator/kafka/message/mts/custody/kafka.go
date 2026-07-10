package custody

import (
	"time"

	"github.com/google/uuid"
)

// This is the orchestrator's own copy of the atlas-mts custody wire contract.
// The orchestrator cannot import the atlas-mts module, so these structs mirror
// services/atlas-mts/atlas.com/mts/kafka/message/custody/kafka.go byte-for-byte
// (identical JSON tags + Type discriminator strings). This follows the
// cashshop/compartment precedent (the orchestrator keeps its own copy of the
// cash-compartment command structs).
const (
	// EnvCommandTopic is the env var naming the MTS custody command topic.
	EnvCommandTopic = "COMMAND_TOPIC_MTS_CUSTODY"

	CommandAcceptToMtsListing      = "ACCEPT_TO_MTS_LISTING"
	CommandReleaseFromMtsHolding   = "RELEASE_FROM_MTS_HOLDING"
	CommandMtsMoveListingToHolding = "MTS_MOVE_LISTING_TO_HOLDING"
	CommandRestoreMtsHolding       = "RESTORE_MTS_HOLDING"
	// CommandRemoveMtsListing hard-deletes a spurious active listing (the
	// late-comp inverse of AcceptToMtsListing).
	CommandRemoveMtsListing = "REMOVE_MTS_LISTING"
	// CommandRestoreListingFromHolding reverses a settlement move: soft-delete the
	// buyer holding and restore the listing to active (the late-comp inverse of
	// MtsMoveListingToHolding).
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
// row in active state.
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

	// offer link
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
// buyer/world identity for the holding to create.
type MtsMoveListingToHoldingCommandBody struct {
	ListingId uuid.UUID `json:"listingId"`
	BuyerId   uint32    `json:"buyerId"`
	WorldId   byte      `json:"worldId"`
}

// RemoveMtsListingCommandBody hard-deletes a spurious active listing by id.
type RemoveMtsListingCommandBody struct {
	ListingId uuid.UUID `json:"listingId"`
}

// RestoreListingFromHoldingCommandBody reverses a settlement move: (listingId,
// buyerId) identify the deterministic buyer holding to soft-delete and the
// listing to transition sold->active.
type RestoreListingFromHoldingCommandBody struct {
	ListingId uuid.UUID `json:"listingId"`
	BuyerId   uint32    `json:"buyerId"`
}

const (
	// EnvStatusEventTopic names the custody status (ack) topic.
	EnvStatusEventTopic = "EVENT_TOPIC_MTS_CUSTODY_STATUS"

	StatusEventTypeAccepted = "ACCEPTED"
	StatusEventTypeReleased = "RELEASED"
	StatusEventTypeMoved    = "MOVED"
	StatusEventTypeRestored = "RESTORED"
	StatusEventTypeError    = "ERROR"
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
