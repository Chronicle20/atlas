package _map

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicMapStatus               topic.Token = "EVENT_TOPIC_MAP_STATUS"
	EventTopicMapStatusTypeCharacterExit             = "CHARACTER_EXIT"
)

// StatusEvent represents a map status event with a generic body
type StatusEvent[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

// CharacterExit represents the body of a character exit event
type CharacterExit struct {
	CharacterId uint32 `json:"characterId"`
}
