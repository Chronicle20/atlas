package message

const (
	ChatTypeGeneral   = "GENERAL"
	ChatTypeBuddy     = "BUDDY"
	ChatTypeParty     = "PARTY"
	ChatTypeGuild     = "GUILD"
	ChatTypeAlliance  = "ALLIANCE"
	ChatTypeWhisper   = "WHISPER"
	ChatTypeMessenger = "MESSENGER"
	ChatTypePet       = "PET"
	ChatTypePinkText  = "PINK_TEXT"
)

const (
	EnvCommandTopicChat = "COMMAND_TOPIC_CHARACTER_CHAT"
)

type Command[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	ActorId   uint32 `json:"actorId"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

const (
	EnvEventTopicChat = "EVENT_TOPIC_CHARACTER_CHAT"
)

type ChatEvent[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	ActorId   uint32 `json:"actorId"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type GeneralChatBody struct {
	BalloonOnly bool `json:"balloonOnly"`
}

type MultiChatBody struct {
	Recipients []uint32 `json:"recipients"`
}

type WhisperChatBody struct {
	Recipient uint32 `json:"recipient"`
}

type WhisperChatEventBody struct {
	RecipientName string `json:"recipientName"`
}

type MessengerChatBody struct {
	Recipients []uint32 `json:"recipients"`
}

type PetChatBody struct {
	OwnerId uint32 `json:"ownerId"`
	PetSlot int8   `json:"petSlot"`
	Type    byte   `json:"type"`
	Action  byte   `json:"action"`
	Balloon bool   `json:"balloon"`
}

type PinkTextChatBody struct {
	Recipients []uint32 `json:"recipients"`
}
