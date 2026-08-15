package pet

import "github.com/google/uuid"

const (
	// EnvCommandTopic defines the environment variable for the pet command topic
	EnvCommandTopic = "COMMAND_TOPIC_PET"
	// CommandTypeAwardCloseness is the command type for awarding closeness to a pet
	CommandTypeAwardCloseness = "AWARD_CLOSENESS"
	// CommandPetEvolve is the command type for evolving a pet
	CommandPetEvolve = "EVOLVE"
	// CommandPetRename is the command type for renaming a pet
	CommandPetRename = "RENAME"

	// Pet status event constants
	EnvEventTopicPetStatus          = "EVENT_TOPIC_PET_STATUS"
	StatusEventTypeClosenessChanged = "CLOSENESS_CHANGED"
	StatusEventTypeEvolved          = "EVOLVED"
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
