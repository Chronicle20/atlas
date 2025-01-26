package fame

const (
	EnvEventTopicFameStatus             = "EVENT_TOPIC_FAME_STATUS"
	StatusEventTypeError                = "ERROR"
	StatusEventErrorTypeNotToday        = "NOT_TODAY"
	StatusEventErrorTypeNotThisMonth    = "NOT_THIS_MONTH"
	StatusEventErrorInvalidName         = "INVALID_NAME"
	StatusEventErrorTypeNotMinimumLevel = "NOT_MINIMUM_LEVEL"
	StatusEventErrorTypeUnexpected      = "UNEXPECTED"
)

type statusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type statusEventErrorBody struct {
	ChannelId byte   `json:"channelId"`
	Error     string `json:"error"`
}
