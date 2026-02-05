package fame

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
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
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type RequestChangeCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	TargetId  uint32     `json:"targetId"`
	Amount    int8       `json:"amount"`
}