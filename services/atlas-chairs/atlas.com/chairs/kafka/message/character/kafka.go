package character

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvEventTopicCharacterStatus           = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeLogin          = "LOGIN"
	EventCharacterStatusTypeLogout         = "LOGOUT"
	EventCharacterStatusTypeChannelChanged = "CHANNEL_CHANGED"
	EventCharacterStatusTypeMapChanged     = "MAP_CHANGED"
)

type StatusEvent[E any] struct {
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	WorldId     world.Id `json:"worldId"`
	Body        E        `json:"body"`
}

type StatusEventLoginBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
}

type StatusEventLogoutBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
}

type StatusEventMapChangedBody struct {
	ChannelId      channel.Id `json:"channelId"`
	OldMapId       _map.Id    `json:"oldMapId"`
	TargetMapId    _map.Id    `json:"targetMapId"`
	TargetPortalId uint32     `json:"targetPortalId"`
}

type ChangeChannelEventLoginBody struct {
	ChannelId    channel.Id `json:"channelId"`
	OldChannelId channel.Id `json:"oldChannelId"`
	MapId        _map.Id    `json:"mapId"`
}
