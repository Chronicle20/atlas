package list

const (
	EnvCommandTopic   = "COMMAND_TOPIC_BUDDY_LIST"
	CommandTypeCreate = "CREATE"

	EnvStatusEventTopic               = "EVENT_TOPIC_BUDDY_LIST_STATUS"
	StatusEventTypeError              = "ERROR"
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

type errorStatusEventBody struct {
	Error string `json:"error"`
}
