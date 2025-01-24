package character

const (
	EnvCommandTopic          = "COMMAND_TOPIC_CHARACTER"
	CommandRequestChangeMeso = "REQUEST_CHANGE_MESO"
	CommandChangeMap         = "CHANGE_MAP"
)

type commandEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type changeMapBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	PortalId  uint32 `json:"portalId"`
}

type requestChangeMesoBody struct {
	Amount int32 `json:"amount"`
}
