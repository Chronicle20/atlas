package portal

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic    = "COMMAND_TOPIC_PORTAL"
	CommandTypeBlock   = "BLOCK"
	CommandTypeUnblock = "UNBLOCK"
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

type BlockBody struct {
	CharacterId uint32 `json:"characterId"`
}

type UnblockBody struct {
	CharacterId uint32 `json:"characterId"`
}
