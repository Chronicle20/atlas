package buddylist

import (
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	// EnvCommandTopic defines the environment variable for the buddy list command topic
	EnvCommandTopic             = "COMMAND_TOPIC_BUDDY_LIST"
	// CommandTypeIncreaseCapacity is the command type for increasing buddy list capacity
	CommandTypeIncreaseCapacity = "INCREASE_CAPACITY"

	// Buddy list status event constants
	EnvEventTopicBuddyListStatus       = "EVENT_TOPIC_BUDDY_LIST_STATUS"
	StatusEventTypeBuddyCapacityUpdate = "CAPACITY_CHANGE"
	StatusEventTypeError               = "ERROR"
)

type Command[E any] struct {
	TransactionId uuid.UUID    `json:"transactionId"`
	WorldId       world.Id     `json:"worldId"`
	CharacterId   character.Id `json:"characterId"`
	Type          string       `json:"type"`
	Body          E            `json:"body"`
}

// IncreaseCapacityCommandBody represents the body of an increase capacity command.
// This command is used to increase a character's buddy list capacity.
type IncreaseCapacityCommandBody struct {
	// NewCapacity is the new capacity value that must be greater than the current capacity
	NewCapacity byte `json:"newCapacity"`
}

// StatusEvent represents a buddy list status event from atlas-buddies
type StatusEvent[E any] struct {
	WorldId     world.Id     `json:"worldId"`
	CharacterId character.Id `json:"characterId"`
	Type        string       `json:"type"`
	Body        E            `json:"body"`
}

// BuddyCapacityChangeStatusEventBody represents the body of a capacity change event
type BuddyCapacityChangeStatusEventBody struct {
	Capacity      byte      `json:"capacity"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
}
