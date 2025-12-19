package note

import (
	"atlas-notes/kafka/message/note"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"time"
)

// CreateNoteStatusEventProvider creates a status event for note creation
func CreateNoteStatusEventProvider(characterId uint32, noteId uint32, senderId uint32, msg string, flag byte, timestamp time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	body := note.StatusEventCreatedBody{
		NoteId:   noteId,
		SenderId: senderId,
		Message:  msg,
		Flag:     flag,
		Time:     timestamp,
	}
	value := note.StatusEvent[note.StatusEventCreatedBody]{
		CharacterId: characterId,
		Type:        note.StatusEventTypeCreated,
		Body:        body,
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
