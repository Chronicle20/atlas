package system_message

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_SYSTEM_MESSAGE"

	CommandSendMessage = "SEND_MESSAGE"
)

// Command represents a Kafka command for system message operations
type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

// SendMessageBody is the body for sending system messages to a character
type SendMessageBody struct {
	MessageType string `json:"messageType"` // "NOTICE", "POP_UP", "PINK_TEXT", "BLUE_TEXT"
	Message     string `json:"message"`
}
