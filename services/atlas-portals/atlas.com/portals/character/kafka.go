package character

import "atlas-portals/tenant"

const (
	EnvEventTopicCharacterStatus        = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeStatChanged = "STAT_CHANGED"

	EnvCommandTopic           = "COMMAND_TOPIC_CHARACTER"
	CommandCharacterChangeMap = "CHANGE_MAP"
)

type statusEvent[E any] struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Type        string       `json:"type"`
	WorldId     byte         `json:"worldId"`
	Body        E            `json:"body"`
}

// TODO this should transmit stats
type statusEventStatChangedBody struct {
	ChannelId       byte `json:"channelId"`
	ExclRequestSent bool `json:"exclRequestSent"`
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
