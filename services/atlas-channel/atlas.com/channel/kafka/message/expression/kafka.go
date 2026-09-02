package expression

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvExpressionCommand topic.Token = "COMMAND_TOPIC_EXPRESSION"
)

type Command struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	CharacterId   uint32     `json:"characterId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	Expression    uint32     `json:"expression"`
	Duration      int32      `json:"duration"`
	ByItemOption  bool       `json:"byItemOption"`
}

const (
	EnvExpressionEvent topic.Token = "EVENT_TOPIC_EXPRESSION"
)

type Event struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	CharacterId   uint32     `json:"characterId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	Expression    uint32     `json:"expression"`
	Duration      int32      `json:"duration"`
	ByItemOption  bool       `json:"byItemOption"`
}
