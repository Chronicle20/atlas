package character

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	_map "github.com/Chronicle20/atlas-constants/map"
)

const (
	EnvEventMemberStatusTopic            = "EVENT_TOPIC_MESSENGER_MEMBER_STATUS"
	EventMessengerMemberStatusTypeLogin  = "LOGIN"
	EventMessengerMemberStatusTypeLogout = "LOGOUT"

	EnvEventTopicCharacterStatus           = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeLogin          = "LOGIN"
	EventCharacterStatusTypeLogout         = "LOGOUT"
	EventCharacterStatusTypeChannelChanged = "CHANNEL_CHANGED"
	EventCharacterStatusTypeMapChanged     = "MAP_CHANGED"
)

type MemberStatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	MessengerId uint32   `json:"messengerId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type MemberLoginEventBody struct {
}

type MemberLogoutEventBody struct {
}

type StatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
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

type StatusEventChannelChangedBody struct {
	ChannelId    channel.Id `json:"channelId"`
	OldChannelId channel.Id `json:"oldChannelId"`
	MapId        _map.Id    `json:"mapId"`
}

type StatusEventMapChangedBody struct {
	ChannelId      channel.Id `json:"channelId"`
	OldMapId       _map.Id    `json:"oldMapId"`
	TargetMapId    _map.Id    `json:"targetMapId"`
	TargetPortalId uint32     `json:"targetPortalId"`
}
