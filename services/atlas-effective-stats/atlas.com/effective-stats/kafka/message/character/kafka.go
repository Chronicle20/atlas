package character

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicCharacterStatus = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeStatChanged   = "STAT_CHANGED"

	EnvCommandTopic = "COMMAND_TOPIC_CHARACTER"
	CommandClampHP  = "CLAMP_HP"
	CommandClampMP  = "CLAMP_MP"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventStatChangedBody struct {
	ChannelId       channel.Id             `json:"channelId"`
	ExclRequestSent bool                   `json:"exclRequestSent"`
	Updates         []string               `json:"updates"`
	Values          map[string]interface{} `json:"values,omitempty"`
}

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type ClampHPBody struct {
	ChannelId channel.Id `json:"channelId"`
	MaxValue  uint16     `json:"maxValue"`
}

type ClampMPBody struct {
	ChannelId channel.Id `json:"channelId"`
	MaxValue  uint16     `json:"maxValue"`
}
