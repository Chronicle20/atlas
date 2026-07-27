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
)

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	PortalId  uint32     `json:"portalId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type EnterBody struct {
	CharacterId uint32 `json:"characterId"`
}

type WarpCommand struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      WarpBody   `json:"body"`
}

type WarpBody struct {
	CharacterId uint32  `json:"characterId"`
	TargetMapId _map.Id `json:"targetMapId"`
	// TargetPortalId: when non-zero, land at this portal in the target map instead
	// of a random spawn point (used to drop a Mystic Door user at the linked door's
	// town portal rather than the town's default spawn).
	TargetPortalId uint32 `json:"targetPortalId"`
	// UseTargetPosition, when true, lands the character at the exact (TargetX,
	// TargetY) coordinate instead of a portal — used by Mystic Door to place the
	// user on the linked door's exact position.
	UseTargetPosition bool  `json:"useTargetPosition"`
	TargetX           int16 `json:"targetX"`
	TargetY           int16 `json:"targetY"`
}
