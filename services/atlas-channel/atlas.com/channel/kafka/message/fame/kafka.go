package fame

const (
	EnvCommandTopic          = "COMMAND_TOPIC_FAME"
	CommandTypeRequestChange = "REQUEST_CHANGE"
)

type Command[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type RequestChangeCommandBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	TargetId  uint32 `json:"targetId"`
	Amount    int8   `json:"amount"`
}

const (
	EnvEventTopicFameStatus             = "EVENT_TOPIC_FAME_STATUS"
	StatusEventTypeError                = "ERROR"
	StatusEventErrorTypeNotToday        = "NOT_TODAY"
	StatusEventErrorTypeNotThisMonth    = "NOT_THIS_MONTH"
	StatusEventErrorInvalidName         = "INVALID_NAME"
	StatusEventErrorTypeNotMinimumLevel = "NOT_MINIMUM_LEVEL"
	StatusEventErrorTypeUnexpected      = "UNEXPECTED"
)

type StatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StatusEventErrorBody struct {
	ChannelId byte   `json:"channelId"`
	Error     string `json:"error"`
}
