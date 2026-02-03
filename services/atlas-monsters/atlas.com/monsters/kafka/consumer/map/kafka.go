package _map

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicMapStatus                = "EVENT_TOPIC_MAP_STATUS"
	EventTopicMapStatusTypeCharacterEnter = "CHARACTER_ENTER"
	EventTopicMapStatusTypeCharacterExit  = "CHARACTER_EXIT"
)

type statusEvent[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type characterEnter struct {
	CharacterId uint32 `json:"characterId"`
}

type characterExit struct {
	CharacterId uint32 `json:"characterId"`
}
