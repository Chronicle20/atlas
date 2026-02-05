package portal_actions

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic  = "COMMAND_TOPIC_PORTAL_ACTIONS"
	CommandTypeEnter = "ENTER"
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

type enterBody struct {
	CharacterId uint32 `json:"characterId"`
	PortalName  string `json:"portalName"`
}
