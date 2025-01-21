package guild

const (
	EnvCommandTopic          = "COMMAND_TOPIC_GUILD"
	CommandTypeRequestCreate = "REQUEST_CREATE"
)

type command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type requestCreateBody struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Name      string `json:"name"`
}
