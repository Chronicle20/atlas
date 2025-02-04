package chair

const (
	EnvCommandTopic    = "COMMAND_TOPIC_CHAIR"
	CommandUseChair    = "USE"
	CommandCancelChair = "CANCEL"

	ChairTypeFixed    = "FIXED"
	ChairTypePortable = "PORTABLE"
)

type command[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type useChairCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	ChairType   string `json:"chairType"`
	ChairId     uint32 `json:"chairId"`
}

type cancelChairCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}
