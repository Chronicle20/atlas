package quest

const (
	// EnvStatusEventTopic defines the environment variable for the quest status event topic
	EnvStatusEventTopic = "EVENT_TOPIC_QUEST_STATUS"

	// Status event types
	StatusEventTypeStarted         = "STARTED"
	StatusEventTypeCompleted       = "COMPLETED"
	StatusEventTypeForfeited       = "FORFEITED"
	StatusEventTypeProgressUpdated = "PROGRESS_UPDATED"
	StatusEventTypeError           = "ERROR"
)

// StatusEvent represents a quest status event message
type StatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
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
