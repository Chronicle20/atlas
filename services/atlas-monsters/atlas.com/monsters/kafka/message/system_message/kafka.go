// Package system_message defines the wire shape for atlas-channel's system
// message commands. The types mirror
// services/atlas-party-quests/atlas.com/party-quests/kafka/message/system_message/kafka.go
// (matching JSON tags) so atlas-monsters can publish SEND_MESSAGE commands
// without importing across a service boundary. This local-copy-per-service
// pattern is the established convention; no shared library is introduced.
package system_message

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic    = "COMMAND_TOPIC_SYSTEM_MESSAGE"
	CommandSendMessage = "SEND_MESSAGE"
)

type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type SendMessageBody struct {
	MessageType string `json:"messageType"`
	Message     string `json:"message"`
}
