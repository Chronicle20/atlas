package message

const (
	EnvCommandTopicGeneralChat = "COMMAND_TOPIC_CHARACTER_GENERAL_CHAT"
	EnvEventTopicGeneralChat   = "EVENT_TOPIC_CHARACTER_GENERAL_CHAT"
)

type generalChatCommand struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
	CharacterId uint32 `json:"characterId"`
	Message     string `json:"message"`
	BalloonOnly bool   `json:"balloonOnly"`
}

type generalChatEvent struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
	CharacterId uint32 `json:"characterId"`
	Message     string `json:"message"`
	BalloonOnly bool   `json:"balloonOnly"`
}
