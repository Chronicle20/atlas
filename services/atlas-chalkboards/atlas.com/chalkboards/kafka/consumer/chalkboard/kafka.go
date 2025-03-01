package character

const (
	EnvCommandTopic        = "COMMAND_TOPIC_CHALKBOARD"
	CommandChalkboardSet   = "SET"
	CommandChalkboardClear = "CLEAR"
)

type commandEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type setCommandBody struct {
	Message string `json:"message"`
}

type clearCommandBody struct {
}
