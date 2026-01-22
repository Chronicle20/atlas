package portal

const (
	EnvPortalCommandTopic = "COMMAND_TOPIC_PORTAL"
	CommandTypeEnter      = "ENTER"
	CommandTypeBlock      = "BLOCK"
	CommandTypeUnblock    = "UNBLOCK"
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

type blockBody struct {
	CharacterId uint32 `json:"characterId"`
}

type unblockBody struct {
	CharacterId uint32 `json:"characterId"`
}
