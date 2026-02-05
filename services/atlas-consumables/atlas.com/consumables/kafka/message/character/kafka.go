package character

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic  = "COMMAND_TOPIC_CHARACTER"
	CommandChangeMap = "CHANGE_MAP"
	CommandChangeHP  = "CHANGE_HP"
	CommandChangeMP  = "CHANGE_MP"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type ChangeHPCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    int16      `json:"amount"`
}

type ChangeMPCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    int16      `json:"amount"`
}

type ChangeMapBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	PortalId  uint32     `json:"portalId"`
}

const (
	EnvEventTopicCharacterStatus           = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeLogin          = "LOGIN"
	EventCharacterStatusTypeLogout         = "LOGOUT"
	EventCharacterStatusTypeChannelChanged = "CHANNEL_CHANGED"
	EventCharacterStatusTypeMapChanged     = "MAP_CHANGED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventLoginBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

type StatusEventLogoutBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

type StatusEventMapChangedBody struct {
	ChannelId      channel.Id `json:"channelId"`
	OldMapId       _map.Id    `json:"oldMapId"`
	OldInstance    uuid.UUID  `json:"oldInstance"`
	TargetMapId    _map.Id    `json:"targetMapId"`
	TargetInstance uuid.UUID  `json:"targetInstance"`
	TargetPortalId uint32     `json:"targetPortalId"`
}

type ChangeChannelEventLoginBody struct {
	ChannelId    channel.Id `json:"channelId"`
	OldChannelId channel.Id `json:"oldChannelId"`
	MapId        _map.Id    `json:"mapId"`
	Instance     uuid.UUID  `json:"instance"`
}
