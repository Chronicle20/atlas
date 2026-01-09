package pet

import "github.com/google/uuid"

const (
	// EnvCommandTopic defines the environment variable for the pet command topic
	EnvCommandTopic = "COMMAND_TOPIC_PET"
	// CommandTypeAwardCloseness is the command type for awarding closeness to a pet
	CommandTypeAwardCloseness = "AWARD_CLOSENESS"

	// Pet status event constants
	EnvEventTopicPetStatus          = "EVENT_TOPIC_PET_STATUS"
	StatusEventTypeClosenessChanged = "CLOSENESS_CHANGED"
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
