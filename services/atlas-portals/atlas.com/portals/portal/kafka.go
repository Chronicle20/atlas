package portal

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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
	CharacterId    uint32  `json:"characterId"`
	TargetMapId    _map.Id `json:"targetMapId"`
	TargetPortalId uint32  `json:"targetPortalId"` // non-zero: land at this portal instead of a random spawn
	// UseTargetPosition, when true, lands the character at the exact (TargetX,
	// TargetY) coordinate instead of a portal — used by Mystic Door.
	UseTargetPosition bool  `json:"useTargetPosition"`
	TargetX           int16 `json:"targetX"`
	TargetY           int16 `json:"targetY"`
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
