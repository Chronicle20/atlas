package expression

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvExpressionEvent   = "EVENT_TOPIC_EXPRESSION"
	EnvExpressionCommand = "COMMAND_TOPIC_EXPRESSION"
)

// StatusEvent represents an expression event
type StatusEvent struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	CharacterId   uint32     `json:"characterId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	Expression    uint32     `json:"expression"`
}

// Command represents an expression command
type Command struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	CharacterId   uint32     `json:"characterId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	Expression    uint32     `json:"expression"`
}