package character

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicCharacterStatus topic.Token = "EVENT_TOPIC_CHARACTER_STATUS"
)

const (
	StatusEventTypeStatChanged = "STAT_CHANGED"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_CHARACTER"
)

const (
	CommandClampHP = "CLAMP_HP"
	CommandClampMP = "CLAMP_MP"
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
	Updates         []stat.Type            `json:"updates"`
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
