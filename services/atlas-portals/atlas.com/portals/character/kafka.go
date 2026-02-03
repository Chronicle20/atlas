package character

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicCharacterStatus        = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeStatChanged = "STAT_CHANGED"

	EnvCommandTopic           = "COMMAND_TOPIC_CHARACTER"
	CommandCharacterChangeMap = "CHANGE_MAP"
)

type statusEvent[E any] struct {
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	WorldId     world.Id `json:"worldId"`
	Body        E        `json:"body"`
}

// TODO this should transmit stats
type statusEventStatChangedBody struct {
	ChannelId       channel.Id `json:"channelId"`
	ExclRequestSent bool       `json:"exclRequestSent"`
}

type commandEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type changeMapBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	PortalId  uint32     `json:"portalId"`
}
