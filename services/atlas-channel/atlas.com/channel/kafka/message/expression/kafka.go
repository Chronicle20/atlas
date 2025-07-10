package expression

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvExpressionCommand = "COMMAND_TOPIC_EXPRESSION"
)

type Command struct {
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	Expression  uint32     `json:"expression"`
}

const (
	EnvExpressionEvent = "EVENT_TOPIC_EXPRESSION"
)

type Event struct {
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	Expression  uint32     `json:"expression"`
}
