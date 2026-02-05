package message

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicChat = "EVENT_TOPIC_CHARACTER_CHAT"

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

type ChatEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	ActorId   uint32     `json:"actorId"`
	Message   string     `json:"message"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
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
