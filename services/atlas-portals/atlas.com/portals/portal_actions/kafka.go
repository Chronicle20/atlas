package portal_actions

const (
	EnvCommandTopic  = "COMMAND_TOPIC_PORTAL_ACTIONS"
	CommandTypeEnter = "ENTER"
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
	PortalName  string `json:"portalName"`
}
