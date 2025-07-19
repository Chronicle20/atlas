package list

const (
	// EnvCommandTopic defines the environment variable for the buddy list command topic
	EnvCommandTopic            = "COMMAND_TOPIC_BUDDY_LIST"
	// CommandTypeCreate is the command type for creating a new buddy list
	CommandTypeCreate          = "CREATE"
	// CommandTypeRequestAdd is the command type for requesting to add a buddy
	CommandTypeRequestAdd      = "REQUEST_ADD"
	// CommandTypeRequestDelete is the command type for requesting to delete a buddy
	CommandTypeRequestDelete   = "REQUEST_DELETE"
	// CommandTypeIncreaseCapacity is the command type for increasing buddy list capacity
	CommandTypeIncreaseCapacity = "INCREASE_CAPACITY"
)

type Command[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type CreateCommandBody struct {
	Capacity byte `json:"capacity"`
}

type RequestAddBuddyCommandBody struct {
	CharacterId   uint32 `json:"characterId"`
	CharacterName string `json:"characterName"`
	Group         string `json:"group"`
}

type RequestDeleteBuddyCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

// IncreaseCapacityCommandBody represents the body of an increase capacity command.
// This command is used to increase a character's buddy list capacity.
type IncreaseCapacityCommandBody struct {
	// NewCapacity is the new capacity value that must be greater than the current capacity
	NewCapacity byte `json:"newCapacity"`
}

const (
	// EnvStatusEventTopic defines the environment variable for the buddy list status event topic
	EnvStatusEventTopic                = "EVENT_TOPIC_BUDDY_LIST_STATUS"
	// StatusEventTypeBuddyAdded is emitted when a buddy is successfully added
	StatusEventTypeBuddyAdded          = "BUDDY_ADDED"
	// StatusEventTypeBuddyRemoved is emitted when a buddy is successfully removed
	StatusEventTypeBuddyRemoved        = "BUDDY_REMOVED"
	// StatusEventTypeBuddyUpdated is emitted when buddy information is updated
	StatusEventTypeBuddyUpdated        = "BUDDY_UPDATED"
	// StatusEventTypeBuddyChannelChange is emitted when a buddy's channel changes
	StatusEventTypeBuddyChannelChange  = "BUDDY_CHANNEL_CHANGE"
	// StatusEventTypeBuddyCapacityUpdate is emitted when buddy list capacity changes
	StatusEventTypeBuddyCapacityUpdate = "CAPACITY_CHANGE"
	// StatusEventTypeError is emitted when an operation fails
	StatusEventTypeError               = "ERROR"

	// StatusEventErrorListFull indicates the requester's buddy list is at capacity
	StatusEventErrorListFull          = "BUDDY_LIST_FULL"
	// StatusEventErrorOtherListFull indicates the target's buddy list is at capacity
	StatusEventErrorOtherListFull     = "OTHER_BUDDY_LIST_FULL"
	// StatusEventErrorAlreadyBuddy indicates the characters are already buddies
	StatusEventErrorAlreadyBuddy      = "ALREADY_BUDDY"
	// StatusEventErrorCannotBuddyGm indicates attempting to buddy a game master
	StatusEventErrorCannotBuddyGm     = "CANNOT_BUDDY_GM"
	// StatusEventErrorCharacterNotFound indicates the character could not be found
	StatusEventErrorCharacterNotFound = "CHARACTER_NOT_FOUND"
	// StatusEventErrorInvalidCapacity indicates the new capacity is invalid (not greater than current)
	StatusEventErrorInvalidCapacity   = "INVALID_CAPACITY"
	// StatusEventErrorUnknownError indicates an unexpected error occurred
	StatusEventErrorUnknownError      = "UNKNOWN_ERROR"
)

type StatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type BuddyAddedStatusEventBody struct {
	CharacterId   uint32 `json:"characterId"`
	Group         string `json:"group"`
	CharacterName string `json:"characterName"`
	ChannelId     int8   `json:"channelId"`
}

type BuddyRemovedStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
}

type BuddyUpdatedStatusEventBody struct {
	CharacterId   uint32 `json:"characterId"`
	Group         string `json:"group"`
	CharacterName string `json:"characterName"`
	ChannelId     int8   `json:"channelId"`
	InShop        bool   `json:"inShop"`
}

type BuddyChannelChangeStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
	ChannelId   int8   `json:"channelId"`
}

type BuddyCapacityChangeStatusEventBody struct {
	Capacity byte `json:"capacity"`
}

type ErrorStatusEventBody struct {
	Error string `json:"error"`
}
