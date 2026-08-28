package system_message

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_SYSTEM_MESSAGE"

	CommandShowHint = "SHOW_HINT"
)

// Command mirrors atlas-channel's system_message command envelope. It is
// duplicated rather than imported: reaching across a service boundary for
// another service's internals is forbidden, and every cross-service contract
// in this repo is carried as a local copy.
type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

// ShowHintBody is the body for showing a hint box to a character.
type ShowHintBody struct {
	Hint   string `json:"hint"`   // Hint text to display
	Width  uint16 `json:"width"`  // Width of the hint box (0 for auto-calculation)
	Height uint16 `json:"height"` // Height of the hint box (0 for auto-calculation)
}
