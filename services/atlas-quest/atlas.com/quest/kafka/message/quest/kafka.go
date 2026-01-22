package quest

import "github.com/google/uuid"

const (
	// EnvCommandTopic defines the environment variable for the quest command topic
	EnvCommandTopic = "COMMAND_TOPIC_QUEST"

	// Command types
	CommandTypeStart          = "START"
	CommandTypeComplete       = "COMPLETE"
	CommandTypeForfeit        = "FORFEIT"
	CommandTypeUpdateProgress = "UPDATE_PROGRESS"
	CommandTypeRestoreItem    = "RESTORE_ITEM"
)

// Command represents a quest command message
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	ChannelId     byte      `json:"channelId"`
	MapId         uint32    `json:"mapId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// StartCommandBody is the body for starting a quest
type StartCommandBody struct {
	QuestId uint32 `json:"questId"`
	NpcId   uint32 `json:"npcId,omitempty"`
	Force   bool   `json:"force,omitempty"` // If true, skip requirement checks
}

// CompleteCommandBody is the body for completing a quest
type CompleteCommandBody struct {
	QuestId   uint32 `json:"questId"`
	NpcId     uint32 `json:"npcId,omitempty"`
	Selection int32  `json:"selection,omitempty"`
	Force     bool   `json:"force,omitempty"` // If true, skip requirement checks and just mark complete
}

// ForfeitCommandBody is the body for forfeiting a quest
type ForfeitCommandBody struct {
	QuestId uint32 `json:"questId"`
}

// UpdateProgressCommandBody is the body for updating quest progress
type UpdateProgressCommandBody struct {
	QuestId    uint32 `json:"questId"`
	InfoNumber uint32 `json:"infoNumber"`
	Progress   string `json:"progress"`
}

// RestoreItemCommandBody is the body for restoring a lost quest item
type RestoreItemCommandBody struct {
	QuestId uint32 `json:"questId"`
	ItemId  uint32 `json:"itemId"`
}

const (
	// EnvStatusEventTopic defines the environment variable for the quest status event topic
	EnvStatusEventTopic = "EVENT_TOPIC_QUEST_STATUS"

	// Status event types
	StatusEventTypeStarted         = "STARTED"
	StatusEventTypeCompleted       = "COMPLETED"
	StatusEventTypeForfeited       = "FORFEITED"
	StatusEventTypeProgressUpdated = "PROGRESS_UPDATED"
	StatusEventTypeError           = "ERROR"

	// Error types
	StatusEventErrorQuestNotFound      = "QUEST_NOT_FOUND"
	StatusEventErrorQuestAlreadyActive = "QUEST_ALREADY_ACTIVE"
	StatusEventErrorQuestNotStarted    = "QUEST_NOT_STARTED"
	StatusEventErrorQuestCompleted     = "QUEST_ALREADY_COMPLETED"
	StatusEventErrorRequirementsNotMet = "REQUIREMENTS_NOT_MET"
	StatusEventErrorUnknownError       = "UNKNOWN_ERROR"
)

// StatusEvent represents a quest status event message
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// QuestStartedEventBody is the body for quest started events
type QuestStartedEventBody struct {
	QuestId uint32 `json:"questId"`
}

// QuestCompletedEventBody is the body for quest completed events
type QuestCompletedEventBody struct {
	QuestId uint32 `json:"questId"`
}

// QuestForfeitedEventBody is the body for quest forfeited events
type QuestForfeitedEventBody struct {
	QuestId uint32 `json:"questId"`
}

// QuestProgressUpdatedEventBody is the body for progress updated events
type QuestProgressUpdatedEventBody struct {
	QuestId    uint32 `json:"questId"`
	InfoNumber uint32 `json:"infoNumber"`
	Progress   string `json:"progress"`
}

// ErrorStatusEventBody is the body for error events
type ErrorStatusEventBody struct {
	QuestId uint32 `json:"questId,omitempty"`
	Error   string `json:"error"`
}
