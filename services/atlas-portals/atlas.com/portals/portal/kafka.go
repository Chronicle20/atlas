package portal

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvPortalCommandTopic = "COMMAND_TOPIC_PORTAL"
	CommandTypeEnter      = "ENTER"
	CommandTypeWarp       = "WARP"
	CommandTypeBlock      = "BLOCK"
	CommandTypeUnblock    = "UNBLOCK"
)

type commandEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	PortalId  uint32     `json:"portalId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type warpEvent struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      warpBody   `json:"body"`
}

type warpBody struct {
	CharacterId uint32  `json:"characterId"`
	TargetMapId _map.Id `json:"targetMapId"`
}

type enterBody struct {
	CharacterId uint32 `json:"characterId"`
}

type blockBody struct {
	CharacterId uint32 `json:"characterId"`
}

type unblockBody struct {
	CharacterId uint32 `json:"characterId"`
}
