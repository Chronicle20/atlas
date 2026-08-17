package buddylist

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	// EnvCommandTopic defines the environment variable for the buddy list command topic
	EnvCommandTopic = "COMMAND_TOPIC_BUDDY_LIST"
	// CommandTypeIncreaseCapacity is the command type for increasing buddy list capacity
	CommandTypeIncreaseCapacity = "INCREASE_CAPACITY"
	// CommandTypeRequestDelete is the command type for requesting to delete a buddy
	CommandTypeRequestDelete = "REQUEST_DELETE"
	// CommandTypeRestore is the command type for restoring a buddy entry a
	// server-issued REQUEST_DELETE removed, in ONE direction and without the
	// invite handshake REQUEST_ADD performs. Mirrors atlas-buddies'
	// CommandTypeRestore. It is the exact inverse of CommandTypeRequestDelete
	// and is emitted twice per pair by the world-transfer saga's compensation
	// (task-227 FR-4.8), matching the 2N REQUEST_DELETEs the severance step
	// sent.
	CommandTypeRestore = "RESTORE"

	// Buddy list status event constants
	EnvEventTopicBuddyListStatus       = "EVENT_TOPIC_BUDDY_LIST_STATUS"
	StatusEventTypeBuddyCapacityUpdate = "CAPACITY_CHANGE"
	StatusEventTypeBuddyRemoved        = "BUDDY_REMOVED"
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

// RequestDeleteBuddyCommandBody represents the body of a request to delete a buddy.
type RequestDeleteBuddyCommandBody struct {
	CharacterId character.Id `json:"characterId"`
}

// RestoreBuddyCommandBody mirrors atlas-buddies' RestoreBuddyCommandBody
// exactly (field names and json tags): the buddy to put back on the commanded
// character's list.
type RestoreBuddyCommandBody struct {
	CharacterId character.Id `json:"characterId"`
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

// BuddyRemovedStatusEventBody represents the body of a buddy removed event
type BuddyRemovedStatusEventBody struct {
	CharacterId   character.Id `json:"characterId"`
	TransactionId uuid.UUID    `json:"transactionId,omitempty"`
}
