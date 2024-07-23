package character

import (
	"atlas-character/tenant"
)

const (
	EnvEventTopicCharacterStatus       = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeCreated    = "CREATED"
	EventCharacterStatusTypeLogin      = "LOGIN"
	EventCharacterStatusTypeLogout     = "LOGOUT"
	EventCharacterStatusTypeMapChanged = "MAP_CHANGED"

	EnvCommandTopic           = "COMMAND_TOPIC_CHARACTER"
	CommandCharacterChangeMap = "CHANGE_MAP"
)

type statusEvent[E any] struct {
	Tenant      tenant.Model `json:"tenant"`
	WorldId     byte         `json:"worldId"`
	CharacterId uint32       `json:"characterId"`
	Type        string       `json:"type"`
	Body        E            `json:"body"`
}

type statusEventCreatedBody struct {
	Name string `json:"name"`
}

type statusEventLoginBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type statusEventLogoutBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type statusEventMapChangedBody struct {
	ChannelId      byte   `json:"channelId"`
	OldMapId       uint32 `json:"oldMapId"`
	TargetMapId    uint32 `json:"targetMapId"`
	TargetPortalId uint32 `json:"targetPortalId"`
}

type commandEvent[E any] struct {
	Tenant      tenant.Model `json:"tenant"`
	WorldId     byte         `json:"worldId"`
	CharacterId uint32       `json:"characterId"`
	Type        string       `json:"type"`
	Body        E            `json:"body"`
}

type changeMapBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	PortalId  uint32 `json:"portalId"`
}
