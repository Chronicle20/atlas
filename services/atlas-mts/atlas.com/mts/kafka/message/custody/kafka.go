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
	ListingId  uuid.UUID `json:"listingId"`
	WorldId    byte      `json:"worldId"`
	SellerId   uint32    `json:"sellerId"`
	SellerName string    `json:"sellerName"`
	SaleType   string    `json:"saleType"`

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
}

// ReleaseFromMtsHoldingCommandBody soft-deletes the take-home holding row.
type ReleaseFromMtsHoldingCommandBody struct {
	HoldingId uuid.UUID `json:"holdingId"`
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

// StatusEventErrorBody reports a custody error.
type StatusEventErrorBody struct {
	Error string `json:"error"`
}
