package character

import "atlas-messages/tenant"

const (
	EnvCommandTopic           = "COMMAND_TOPIC_CHARACTER"
	CommandCharacterChangeMap = "CHANGE_MAP"
)

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
