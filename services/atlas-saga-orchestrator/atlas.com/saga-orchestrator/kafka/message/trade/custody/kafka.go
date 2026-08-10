package custody

import (
	"github.com/google/uuid"
)

// This is the orchestrator's own copy of the atlas-trades custody wire
// contract. The orchestrator cannot import the atlas-trades module, so these
// structs mirror
// services/atlas-trades/atlas.com/trades/kafka/message/custody/kafka.go
// byte-for-byte (identical JSON tags + Type discriminator strings). This
// follows the mts/custody and cashshop/compartment precedent.
//
// Trade escrow exists because a staged item must genuinely LEAVE its owner's
// compartment (task-205 design §5A): the resulting inventory delta is what
// clears the client's m_bExclRequestSent, and nothing else in the trade flow
// does. The escrow row is the durable custody record that makes returning the
// item possible without knowing which saga staged it.
const (
	// EnvCommandTopic is the env var naming the trade custody command topic.
	EnvCommandTopic = "COMMAND_TOPIC_TRADE_CUSTODY"

	CommandAcceptToTrade    = "ACCEPT_TO_TRADE"
	CommandReleaseFromTrade = "RELEASE_FROM_TRADE"
	// CommandRestoreTradeEscrow un-soft-deletes an escrow row (the late-comp
	// inverse of ReleaseFromTrade).
	CommandRestoreTradeEscrow = "RESTORE_TRADE_ESCROW"
	// CommandRemoveTradeEscrow hard-deletes a spurious escrow row (the
	// late-comp inverse of AcceptToTrade).
	CommandRemoveTradeEscrow = "REMOVE_TRADE_ESCROW"
)

// Command is the generic custody command envelope. TransactionId keys the saga
// step; Type discriminates which body is carried.
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// AcceptToTradeCommandBody carries every field needed to CREATE an escrow row.
//
// The stat block is spelled out rather than nested so the row can be written
// with explicit name-keyed columns — the same COPY/restore column-order
// discipline atlas-mts's holdings table follows.
type AcceptToTradeCommandBody struct {
	EscrowId            uuid.UUID `json:"escrowId"`
	RoomId              uuid.UUID `json:"roomId"`
	OwnerId             uint32    `json:"ownerId"`
	TradeSlot           byte      `json:"tradeSlot"`
	SourceInventoryType byte      `json:"sourceInventoryType"`
	SourceSlot          int16     `json:"sourceSlot"`

	// item snapshot
	TemplateId    uint32 `json:"templateId"`
	Quantity      uint32 `json:"quantity"`
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
	Owner         string `json:"owner"`
}

// ReleaseFromTradeCommandBody soft-deletes the escrow row. The row holds the
// snapshot, so a release can never disagree with the accept that created it.
type ReleaseFromTradeCommandBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
}

// RestoreTradeEscrowCommandBody un-soft-deletes the escrow row by id.
type RestoreTradeEscrowCommandBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
}

// RemoveTradeEscrowCommandBody hard-deletes a spurious escrow row by id.
type RemoveTradeEscrowCommandBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
}

const (
	// EnvStatusEventTopic names the trade custody status (ack) topic.
	EnvStatusEventTopic = "EVENT_TOPIC_TRADE_CUSTODY_STATUS"

	StatusEventTypeAccepted = "ACCEPTED"
	StatusEventTypeReleased = "RELEASED"
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

// StatusEventAcceptedBody acks escrow creation, echoing the escrow id.
type StatusEventAcceptedBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
}

// StatusEventReleasedBody acks an escrow release, echoing the escrow id.
type StatusEventReleasedBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
}

// StatusEventRestoredBody acks an escrow restore, echoing the escrow id.
type StatusEventRestoredBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
}

// StatusEventErrorBody reports a custody failure with a human-readable reason.
type StatusEventErrorBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
	Error    string    `json:"error"`
}
