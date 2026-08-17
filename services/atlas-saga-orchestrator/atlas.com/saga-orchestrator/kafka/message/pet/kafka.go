package pet

import (
	"time"

	"github.com/google/uuid"
)

const (
	// EnvCommandTopic defines the environment variable for the pet command topic
	EnvCommandTopic = "COMMAND_TOPIC_PET"
	// CommandTypeAwardCloseness is the command type for awarding closeness to a pet
	CommandTypeAwardCloseness = "AWARD_CLOSENESS"
	// CommandPetEvolve is the command type for evolving a pet
	CommandPetEvolve = "EVOLVE"
	// CommandPetRevive is the command type for reviving a dried-up pet
	CommandPetRevive = "REVIVE"
	// CommandPetRename is the command type for renaming a pet
	CommandPetRename = "RENAME"

	// Pet status event constants
	EnvEventTopicPetStatus          = "EVENT_TOPIC_PET_STATUS"
	StatusEventTypeClosenessChanged = "CLOSENESS_CHANGED"
	StatusEventTypeEvolved          = "EVOLVED"
	StatusEventTypeRevived          = "REVIVED"
	StatusEventTypeReviveFailed     = "REVIVE_FAILED"
	StatusEventTypeNameChanged      = "NAME_CHANGED"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	ActorId       uint32    `json:"actorId"`
	PetId         uint32    `json:"petId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// AwardClosenessCommandBody represents the body of an award closeness command.
// This command is used to increase a pet's closeness level.
type AwardClosenessCommandBody struct {
	// Amount is the amount of closeness to add to the pet
	Amount uint16 `json:"amount"`
}

// EvolveCommandBody represents the body of an evolve command.
// This command is used to evolve a pet. It carries no additional fields.
type EvolveCommandBody struct{}

// ReviveCommandBody restores a dried-up pet's lifespan. It carries NO
// expiration: atlas-pets derives it from the consumed item's own WZ data, so a
// forged command cannot dictate a lifespan. SourceTemplateId names the consumed
// Water of Life (classification 518). Command[E] already carries TransactionId,
// ActorId and PetId, so the body needs nothing else.
type ReviveCommandBody struct {
	SourceTemplateId uint32 `json:"sourceTemplateId"`
}

// RenameCommandBody carries the new pet name. It is ALREADY normalized by the
// caller, but atlas-pets re-validates it regardless (PRD FR-5.6) — the channel
// is not trusted to have validated, and a crafted producer could publish
// straight to this topic.
type RenameCommandBody struct {
	Name string `json:"name"`
}

// StatusEvent represents a pet status event from atlas-pets
type StatusEvent[E any] struct {
	PetId   uint32 `json:"petId"`
	OwnerId uint32 `json:"ownerId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

// ClosenessChangedStatusEventBody represents the body of a closeness changed event
type ClosenessChangedStatusEventBody struct {
	Slot          int8      `json:"slot"`
	Closeness     uint16    `json:"closeness"`
	Amount        int16     `json:"amount"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
}

// EvolvedStatusEventBody represents the body of a pet evolved event.
type EvolvedStatusEventBody struct {
	Slot          int8      `json:"slot"`
	OldTemplateId uint32    `json:"oldTemplateId"`
	NewTemplateId uint32    `json:"newTemplateId"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
}

// RevivedStatusEventBody reports a successful Water of Life revive. Expiration
// is the absolute new dry-up instant; Slot is unchanged by the revive (a doll
// stays unsummoned) and is carried only so consumers need no extra read.
type RevivedStatusEventBody struct {
	Slot          int8      `json:"slot"`
	Expiration    time.Time `json:"expiration"`
	TransactionId uuid.UUID `json:"transactionId"`
}

// ReviveFailedStatusEventBody is a REAL terminal failure event, not a silent
// drop. By the time REVIVE runs the player's Water of Life is already
// destroyed by the saga's first step, so a timeout-length wait for the refund
// would read as a lost item; the saga accepts this event and compensates
// immediately.
type ReviveFailedStatusEventBody struct {
	Reason        string    `json:"reason"`
	TransactionId uuid.UUID `json:"transactionId"`
}

// NameChangedStatusEventBody drives two consumers. atlas-channel needs Slot to
// address the clientbound PET_NAMECHANGE packet (the packet carries no pet id —
// it is routed by ownerId+slot). The orchestrator needs TransactionId to
// complete the rename_pet step. PreviousName is what the compensator re-applies
// if the consume step later fails.
type NameChangedStatusEventBody struct {
	Slot          int8      `json:"slot"`
	Name          string    `json:"name"`
	PreviousName  string    `json:"previousName"`
	TransactionId uuid.UUID `json:"transactionId"`
}
