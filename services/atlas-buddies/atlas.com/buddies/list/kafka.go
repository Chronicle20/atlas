package list

const (
	EnvCommandTopic   = "COMMAND_TOPIC_BUDDY_LIST"
	CommandTypeCreate = "CREATE"

	EnvStatusEventTopic                = "EVENT_TOPIC_BUDDY_LIST_STATUS"
	StatusEventTypeBuddyAdded          = "BUDDY_ADDED"
	StatusEventTypeBuddyRemoved        = "BUDDY_REMOVED"
	StatusEventTypeBuddyChannelChange  = "BUDDY_CHANNEL_CHANGE"
	StatusEventTypeBuddyCapacityUpdate = "CAPACITY_CHANGE"
	StatusEventTypeError               = "ERROR"

	StatusEventErrorListFull          = "BUDDY_LIST_FULL"
	StatusEventErrorOtherListFull     = "OTHER_BUDDY_LIST_FULL"
	StatusEventErrorAlreadyBuddy      = "ALREADY_BUDDY"
	StatusEventErrorCannotBuddyGm     = "CANNOT_BUDDY_GM"
	StatusEventErrorCharacterNotFound = "CHARACTER_NOT_FOUND"
	StatusEventErrorUnknownError      = "UNKNOWN_ERROR"
)

type command[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type createCommandBody struct {
	Capacity byte `json:"capacity"`
}

type statusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type buddyAddedStatusEventBody struct {
	CharacterId   uint32 `json:"characterId"`
	Group         string `json:"group"`
	CharacterName string `json:"characterName"`
	ChannelId     int8   `json:"channelId"`
}

type buddyRemovedStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
}

type buddyChannelChangeStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
	ChannelId   int8   `json:"channelId"`
}

type buddyCapacityChangeStatusEventBody struct {
	Capacity byte `json:"capacity"`
}

type errorStatusEventBody struct {
	Error string `json:"error"`
}
