package portal

const (
	EnvCommandTopic    = "COMMAND_TOPIC_PORTAL"
	CommandTypeBlock   = "BLOCK"
	CommandTypeUnblock = "UNBLOCK"
)

type Command[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	PortalId  uint32 `json:"portalId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type BlockBody struct {
	CharacterId uint32 `json:"characterId"`
}

type UnblockBody struct {
	CharacterId uint32 `json:"characterId"`
}
