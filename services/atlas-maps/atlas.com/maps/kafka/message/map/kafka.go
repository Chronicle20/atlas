package _map

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
)

const (
	EnvEventTopicMapStatus                = "EVENT_TOPIC_MAP_STATUS"
	EventTopicMapStatusTypeCharacterEnter = "CHARACTER_ENTER"
	EventTopicMapStatusTypeCharacterExit  = "CHARACTER_EXIT"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type CharacterEnter struct {
	CharacterId uint32 `json:"characterId"`
}

type CharacterExit struct {
	CharacterId uint32 `json:"characterId"`
}