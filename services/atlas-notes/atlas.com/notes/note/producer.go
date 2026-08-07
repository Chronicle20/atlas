package note

import (
	"atlas-notes/kafka/message/note"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// CreateNoteStatusEventProvider creates a status event for note creation.
// transactionId is uuid.Nil for non-saga creations (REST).
func CreateNoteStatusEventProvider(transactionId uuid.UUID, characterId uint32, noteId uint32, senderId uint32, msg string, flag byte, timestamp time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	body := note.StatusEventCreatedBody{
		NoteId:   noteId,
		SenderId: senderId,
		Message:  msg,
		Flag:     flag,
		Time:     timestamp,
	}
	value := note.StatusEvent[note.StatusEventCreatedBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          note.StatusEventTypeCreated,
		Body:          body,
	}
	return producer.SingleMessageProvider(key, value)
}

// UpdateNoteStatusEventProvider creates a status event for note update
func UpdateNoteStatusEventProvider(characterId uint32, noteId uint32, senderId uint32, msg string, flag byte, timestamp time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	body := note.StatusEventUpdatedBody{
		NoteId:   noteId,
		SenderId: senderId,
		Message:  msg,
		Flag:     flag,
		Time:     timestamp,
	}
	value := note.StatusEvent[note.StatusEventUpdatedBody]{
		CharacterId: characterId,
		Type:        note.StatusEventTypeUpdated,
		Body:        body,
	}
	return producer.SingleMessageProvider(key, value)
}

// DeleteNoteStatusEventProvider creates a status event for note deletion
func DeleteNoteStatusEventProvider(characterId uint32, noteId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	body := note.StatusEventDeletedBody{
		NoteId: noteId,
	}
	value := note.StatusEvent[note.StatusEventDeletedBody]{
		CharacterId: characterId,
		Type:        note.StatusEventTypeDeleted,
		Body:        body,
	}
	return producer.SingleMessageProvider(key, value)
}

// CreateFailedStatusEventProvider creates a status event for a failed note
// creation, so the saga orchestrator can fail the create_note step and
// compensate the consumed Note item.
func CreateFailedStatusEventProvider(transactionId uuid.UUID, characterId uint32, senderId uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := note.StatusEvent[note.StatusEventCreateFailedBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          note.StatusEventTypeCreateFailed,
		Body: note.StatusEventCreateFailedBody{
			SenderId: senderId,
			Reason:   reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
