package note

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic         = "COMMAND_TOPIC_NOTE"
	EnvEventTopicNoteStatus = "EVENT_TOPIC_NOTE_STATUS"

	CommandTypeCreate  = "CREATE"
	CommandTypeDiscard = "DISCARD"

	StatusEventTypeCreated      = "CREATED"
	StatusEventTypeUpdated      = "UPDATED"
	StatusEventTypeDeleted      = "DELETED"
	StatusEventTypeCreateFailed = "CREATE_FAILED"
)

// Command represents a Kafka command for note operations
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
	// acknowledgement; its fame was settled at acceptance time, so Discard
	// must not also fame the sender.
	GiftNote bool `json:"giftNote,omitempty"`
}

// CommandDiscardBody contains data for discarding notes
type CommandDiscardBody struct {
	NoteIds []uint32 `json:"noteIds"`
}

// StatusEvent represents a Kafka status event for note operations
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId,omitempty"` // Saga transaction id (uuid.Nil when not saga-driven)
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// StatusEventCreatedBody contains data for a note created event
type StatusEventCreatedBody struct {
	NoteId   uint32    `json:"noteId"`
	SenderId uint32    `json:"senderId"`
	Message  string    `json:"message"`
	Flag     byte      `json:"flag"`
	Time     time.Time `json:"time"`
}

// StatusEventUpdatedBody contains data for a note updated event
type StatusEventUpdatedBody struct {
	NoteId   uint32    `json:"noteId"`
	SenderId uint32    `json:"senderId"`
	Message  string    `json:"message"`
	Flag     byte      `json:"flag"`
	Time     time.Time `json:"time"`
}

// StatusEventDeletedBody contains data for a note deleted event
type StatusEventDeletedBody struct {
	NoteId uint32 `json:"noteId"`
}

// StatusEventCreateFailedBody contains data for a note create failure event
type StatusEventCreateFailedBody struct {
	SenderId uint32 `json:"senderId"`
	Reason   string `json:"reason"`
}
