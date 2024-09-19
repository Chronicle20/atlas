package portal

const (
	EnvPortalCommandTopic = "COMMAND_TOPIC_PORTAL"
	CommandTypeEnter      = "ENTER"
)

type commandEvent[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	PortalId  uint32 `json:"portalId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type enterBody struct {
	CharacterId uint32 `json:"characterId"`
}
