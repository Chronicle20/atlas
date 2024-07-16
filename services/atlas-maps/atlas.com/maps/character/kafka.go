package character

import (
	"atlas-maps/tenant"
)

const (
	EnvEventTopicCharacterStatus   = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeLogin  = "LOGIN"
	EventCharacterStatusTypeLogout = "LOGOUT"
)

type statusEvent[E any] struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Type        string       `json:"type"`
	WorldId     byte         `json:"worldId"`
	Body        E            `json:"body"`
}

type statusEventLoginBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type statusEventLogoutBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}
