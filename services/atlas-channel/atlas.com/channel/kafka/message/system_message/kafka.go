package system_message

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_SYSTEM_MESSAGE"

	CommandSendMessage     = "SEND_MESSAGE"
	CommandPlayPortalSound = "PLAY_PORTAL_SOUND"
	CommandShowInfo        = "SHOW_INFO"
	CommandShowInfoText    = "SHOW_INFO_TEXT"
	CommandUpdateAreaInfo  = "UPDATE_AREA_INFO"
	CommandShowHint        = "SHOW_HINT"
	CommandShowGuideHint   = "SHOW_GUIDE_HINT"
	CommandShowIntro       = "SHOW_INTRO"
	CommandFieldEffect     = "FIELD_EFFECT"
)

// Command represents a Kafka command for system message operations
type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

// SendMessageBody is the body for sending system messages to a character
type SendMessageBody struct {
	MessageType string `json:"messageType"` // "NOTICE", "POP_UP", "PINK_TEXT", "BLUE_TEXT"
	Message     string `json:"message"`
}

// PlayPortalSoundBody is the body for playing the portal sound effect
// This is an empty struct as no additional data is needed
type PlayPortalSoundBody struct {
}

// ShowInfoBody is the body for showing info/tutorial effects to a character
type ShowInfoBody struct {
	Path string `json:"path"` // Path to the info effect (e.g., "Effect/OnUserEff.img/RecoveryUp")
}

// ShowInfoTextBody is the body for showing text messages to a character
type ShowInfoTextBody struct {
	Text string `json:"text"` // Text message to display
}

// UpdateAreaInfoBody is the body for updating area info (quest record ex) for a character
type UpdateAreaInfoBody struct {
	Area uint16 `json:"area"` // Area/info number (questId in the protocol)
	Info string `json:"info"` // Info string to display
}

// ShowHintBody is the body for showing a hint box to a character
type ShowHintBody struct {
	Hint   string `json:"hint"`   // Hint text to display
	Width  uint16 `json:"width"`  // Width of the hint box (0 for auto-calculation)
	Height uint16 `json:"height"` // Height of the hint box (0 for auto-calculation)
}

// ShowGuideHintBody is the body for showing a pre-defined guide hint by ID to a character
type ShowGuideHintBody struct {
	HintId   uint32 `json:"hintId"`   // Pre-defined hint ID (maps to client's guide hint system)
	Duration uint32 `json:"duration"` // Duration in milliseconds (default 7000ms if 0)
}

// ShowIntroBody is the body for showing an intro/direction effect to a character
type ShowIntroBody struct {
	Path string `json:"path"` // Path to the intro effect (e.g., "Effect/Direction1.img/aranTutorial/ClickPoleArm")
}

// FieldEffectBody is the body for showing a field effect to a character
type FieldEffectBody struct {
	Path string `json:"path"` // Path to the field effect (e.g., "maplemap/enter/1020000")
}
