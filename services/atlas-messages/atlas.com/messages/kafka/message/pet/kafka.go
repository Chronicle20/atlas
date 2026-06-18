package pet

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopic       = "COMMAND_TOPIC_PET"
	CommandAwardCloseness = "AWARD_CLOSENESS"
)

// Command is the pet command envelope consumed by atlas-pets.
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	ActorId       uint32    `json:"actorId"`
	PetId         uint32    `json:"petId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// AwardClosenessCommandBody is the body of an AWARD_CLOSENESS command (additive).
type AwardClosenessCommandBody struct {
	Amount uint16 `json:"amount"`
}
