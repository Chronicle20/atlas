package custody

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"

	"github.com/google/uuid"
)

// This is the orchestrator's own copy of the atlas-parcel custody wire
// contract. The orchestrator cannot import the atlas-parcel module, so these
// structs mirror atlas-parcel's kafka/message/custody/kafka.go byte-for-byte
// (identical JSON tags + Type discriminator strings). Mirrors the MTS custody
// twin (kafka/message/mts/custody/kafka.go).
const (
	// EnvCommandTopic is the env var naming the parcel custody command topic.
	EnvCommandTopic = "COMMAND_TOPIC_PARCEL_CUSTODY"

	CommandAcceptToParcel    = "ACCEPT_TO_PARCEL"
	CommandReleaseFromParcel = "RELEASE_FROM_PARCEL"
	// CommandRestoreParcel un-resolves a parcel released by a
	// withdraw_from_parcel whose accept_to_character then failed (otherwise
	// the item is lost). The late-comp inverse of RELEASE_FROM_PARCEL.
	CommandRestoreParcel = "RESTORE_PARCEL"
	// CommandRemoveParcel hard-deletes a still-pending parcel row created by a
	// late accept_to_parcel after its saga already compensated (otherwise the
	// item is duplicated). The late-comp inverse of ACCEPT_TO_PARCEL.
	CommandRemoveParcel = "REMOVE_PARCEL"
)

// Command is the generic custody command envelope. TransactionId keys the saga
// step; Type discriminates which body is carried.
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// AcceptToParcelCommandBody carries every field needed to CREATE a parcel row
// in custody: sender/recipient identity, delivery parameters, and the full
// item snapshot (looked up from inventory during expansion). HasItem is the
// meso-only escape hatch: false leaves the whole snapshot zero-valued and
// atlas-parcel creates the row with a nil item id.
type AcceptToParcelCommandBody struct {
	ParcelId           uuid.UUID `json:"parcelId"`
	CharacterId        uint32    `json:"characterId"`
	WorldId            world.Id  `json:"worldId"`
	SenderAccountId    uint32    `json:"senderAccountId"`
	SenderName         string    `json:"senderName"`
	RecipientId        uint32    `json:"recipientId"`
	RecipientAccountId uint32    `json:"recipientAccountId"`
	MesoAmount         uint32    `json:"mesoAmount"`
	FeePaid            uint32    `json:"feePaid"`
	Quick              bool      `json:"quick"`
	Message            string    `json:"message"`
	ReceivableAt       time.Time `json:"receivableAt"`
	ExpiresAt          time.Time `json:"expiresAt"`

	HasItem bool `json:"hasItem"`

	// Item snapshot
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

// ReleaseFromParcelCommandBody transitions the parcel row to received AND
// releases custody in one atlas-parcel transaction — the status change and
// the custody release are the same fact and must not become two steps that
// can disagree.
type ReleaseFromParcelCommandBody struct {
	ParcelId    uuid.UUID `json:"parcelId"`
	RecipientId uint32    `json:"recipientId"`
}

// RestoreParcelCommandBody un-resolves a parcel row by id (the compensating
// inverse of RELEASE_FROM_PARCEL).
type RestoreParcelCommandBody struct {
	ParcelId uuid.UUID `json:"parcelId"`
}

// RemoveParcelCommandBody hard-deletes a still-pending parcel row by id (the
// compensating inverse of ACCEPT_TO_PARCEL).
type RemoveParcelCommandBody struct {
	ParcelId uuid.UUID `json:"parcelId"`
}

const (
	// EnvStatusTopic names the parcel custody status (ack) topic.
	EnvStatusTopic = "EVENT_TOPIC_PARCEL_CUSTODY_STATUS"

	StatusEventAccepted = "ACCEPTED"
	StatusEventReleased = "RELEASED"
	StatusEventError    = "ERROR"
)

// StatusEvent is the generic custody ack envelope. TransactionId echoes the
// command so the orchestrator can complete/fail the saga step.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// StatusEventAcceptedBody acks parcel creation, echoing the parcel id.
type StatusEventAcceptedBody struct {
	ParcelId uuid.UUID `json:"parcelId"`
}

// StatusEventReleasedBody acks a parcel release, echoing the parcel id.
type StatusEventReleasedBody struct {
	ParcelId uuid.UUID `json:"parcelId"`
}

// StatusEventErrorBody reports a custody error.
type StatusEventErrorBody struct {
	Error string `json:"error"`
}
