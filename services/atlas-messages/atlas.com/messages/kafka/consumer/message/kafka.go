package message

const (
	EnvCommandTopicChat = "COMMAND_TOPIC_CHARACTER_CHAT"

	ChatTypeGeneral   = "GENERAL"
	ChatTypeBuddy     = "BUDDY"
	ChatTypeParty     = "PARTY"
	ChatTypeGuild     = "GUILD"
	ChatTypeAlliance  = "ALLIANCE"
	ChatTypeWhisper   = "WHISPER"
	ChatTypeMessenger = "MESSENGER"
	ChatTypePet       = "PET"
)

type chatCommand[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	ActorId   uint32 `json:"actorId"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type generalChatBody struct {
	BalloonOnly bool `json:"balloonOnly"`
}

type multiChatBody struct {
	Recipients []uint32 `json:"recipients"`
}

type whisperChatBody struct {
	RecipientName string `json:"recipientName"`
}

type messengerChatBody struct {
	Recipients []uint32 `json:"recipients"`
}

type petChatBody struct {
	OwnerId uint32 `json:"ownerId"`
	PetSlot int8   `json:"petSlot"`
	Type    byte   `json:"type"`
	Action  byte   `json:"action"`
	Balloon bool   `json:"balloon"`
}
