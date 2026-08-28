package note

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic         topic.Token = "COMMAND_TOPIC_NOTE"
	EnvEventTopicNoteStatus topic.Token = "EVENT_TOPIC_NOTE_STATUS"
)

const (
	CommandTypeCreate = "CREATE"

	StatusEventTypeCreated      = "CREATED"
	StatusEventTypeCreateFailed = "CREATE_FAILED"
)

// Command mirrors atlas-notes' note command envelope. WorldId/ChannelId are
// unused by the CREATE handler (atlas-notes kafka/consumer/note/consumer.go)
// and are zero for orchestrator-emitted commands.
type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId,omitempty"` // Saga transaction id (uuid.Nil when not saga-driven)
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

// CommandCreateBody contains data for creating a note
type CommandCreateBody struct {
	SenderId uint32 `json:"senderId"`
	Message  string `json:"message"`
	Flag     byte   `json:"flag"`
	// GiftNote records that this note originated from a cash-shop gift
	// acknowledgement; its fame was settled at acceptance time.
	GiftNote bool `json:"giftNote,omitempty"`
}

// StatusEvent mirrors atlas-notes' note status event envelope.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId,omitempty"` // Saga transaction id (uuid.Nil when not saga-driven)
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// StatusEventCreatedBody contains data for a note created event. Only the
// fields the orchestrator matches on are declared; extra fields on the wire
// are ignored by encoding/json.
type StatusEventCreatedBody struct {
	NoteId   uint32 `json:"noteId"`
	SenderId uint32 `json:"senderId"`
}

// StatusEventCreateFailedBody contains data for a note create failure event
type StatusEventCreateFailedBody struct {
	SenderId uint32 `json:"senderId"`
	Reason   string `json:"reason"`
}
