package message

const (
	EnvEventTopicChat = "EVENT_TOPIC_CHARACTER_CHAT"

	ChatTypeGeneral   = "GENERAL"
	ChatTypeBuddy     = "BUDDY"
	ChatTypeParty     = "PARTY"
	ChatTypeGuild     = "GUILD"
	ChatTypeAlliance  = "ALLIANCE"
	ChatTypeWhisper   = "WHISPER"
	ChatTypeMessenger = "MESSENGER"
)

type chatEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
	CharacterId uint32 `json:"characterId"`
	Message     string `json:"message"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type generalChatBody struct {
	BalloonOnly bool `json:"balloonOnly"`
}

type multiChatBody struct {
	Recipients []uint32 `json:"recipients"`
}

type whisperChatBody struct {
	Recipient uint32 `json:"recipient"`
}

type messengerChatBody struct {
	Recipients []uint32 `json:"recipients"`
}
