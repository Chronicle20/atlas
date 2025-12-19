package fame

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicFameStatus             = "EVENT_TOPIC_FAME_STATUS"
	StatusEventTypeError                = "ERROR"
	StatusEventErrorTypeNotToday        = "NOT_TODAY"
	StatusEventErrorTypeNotThisMonth    = "NOT_THIS_MONTH"
	StatusEventErrorInvalidName         = "INVALID_NAME"
	StatusEventErrorTypeNotMinimumLevel = "NOT_MINIMUM_LEVEL"
	StatusEventErrorTypeUnexpected      = "UNEXPECTED"

	EnvCommandTopic          = "COMMAND_TOPIC_FAME"
	CommandTypeRequestChange = "REQUEST_CHANGE"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type StatusEventErrorBody struct {
	ChannelId channel.Id `json:"channelId"`
	Error     string     `json:"error"`
}

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type RequestChangeCommandBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	TargetId  uint32 `json:"targetId"`
	Amount    int8   `json:"amount"`
}