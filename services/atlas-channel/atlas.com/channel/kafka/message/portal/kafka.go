package portal

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvPortalCommandTopic = "COMMAND_TOPIC_PORTAL"
	CommandTypeEnter      = "ENTER"
)

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	PortalId  uint32     `json:"portalId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type EnterBody struct {
	CharacterId uint32 `json:"characterId"`
}
