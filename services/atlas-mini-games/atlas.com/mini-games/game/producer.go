package game

import (
	"atlas-mini-games/kafka/message/minigame"
	"atlas-mini-games/record"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// recordBody projects a persisted win/tie/loss record onto the wire body.
func recordBody(m record.Model) minigame.RecordBody {
	return minigame.RecordBody{
		GameType: string(m.GameType()),
		Wins:     m.Wins(),
		Ties:     m.Ties(),
		Losses:   m.Losses(),
	}
}

// statusEventProvider builds a keyed (by mapId) status event provider. Every
// event carries the room identity (roomId/ownerId/visitorId) plus the acting
// character id.
func statusEventProvider[E any](transactionId uuid.UUID, f field.Model, roomId uint32, ownerId uint32, visitorId uint32, characterId uint32, eventType string, body E) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &minigame.StatusEvent[E]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		RoomId:        roomId,
		OwnerId:       ownerId,
		VisitorId:     visitorId,
		CharacterId:   characterId,
		Type:          eventType,
		Body:          body,
	}
	return producer.SingleMessageProvider(key, value)
}

// createErrorProvider announces a CREATE_ERROR carrying an enterError key.
func createErrorProvider(transactionId uuid.UUID, f field.Model, characterId uint32, code string) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, f, 0, 0, 0, characterId, minigame.EventTypeCreateError, minigame.ErrorEventBody{Code: code})
}

// enterErrorProvider announces an ENTER_ERROR carrying an enterError key. roomId
// is echoed so the client can correlate the failed visit.
func enterErrorProvider(transactionId uuid.UUID, f field.Model, roomId uint32, characterId uint32, code string) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, f, roomId, roomId, 0, characterId, minigame.EventTypeEnterError, minigame.ErrorEventBody{Code: code})
}

func createdProvider(transactionId uuid.UUID, r Room, ownerRecord record.Model) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), 0, r.OwnerId(), minigame.EventTypeCreated, minigame.CreatedEventBody{
		RoomType:    r.RoomType(),
		Title:       r.Title(),
		PieceType:   r.PieceType(),
		OwnerRecord: recordBody(ownerRecord),
	})
}

func enteredProvider(transactionId uuid.UUID, r Room, ownerRecord record.Model, visitorRecord record.Model) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), r.VisitorId(), minigame.EventTypeEntered, minigame.EnteredEventBody{
		Slot:          1,
		RoomType:      r.RoomType(),
		Title:         r.Title(),
		PieceType:     r.PieceType(),
		OwnerRecord:   recordBody(ownerRecord),
		VisitorRecord: recordBody(visitorRecord),
		OwnerScore:    r.OwnerScore(),
		VisitorScore:  r.VisitorScore(),
	})
}

// leftProvider announces a LEFT event for slot (0 owner, 1 visitor) with the
// given leave status (4 left / 5 expelled). characterId is the character that
// occupied the slot.
func leftProvider(transactionId uuid.UUID, r Room, slot byte, status byte, characterId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), characterId, characterId, minigame.EventTypeLeft, minigame.LeftEventBody{
		Slot:   slot,
		Status: status,
	})
}

// roomClosedProvider announces a ROOM_CLOSED event to the room, carrying the
// visitor's leave status (3 closed).
func roomClosedProvider(transactionId uuid.UUID, r Room, visitorStatus byte) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), r.OwnerId(), minigame.EventTypeRoomClosed, minigame.RoomClosedEventBody{
		VisitorStatus: visitorStatus,
	})
}

func chatProvider(transactionId uuid.UUID, r Room, slot byte, characterId uint32, msg string) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), characterId, minigame.EventTypeChat, minigame.ChatEventBody{
		Slot:    slot,
		Message: msg,
	})
}

// balloonProvider announces a BALLOON_UPDATED event for the room's field. occupancy
// is the current head-count (1 owner-only, 2 both); remove tears the balloon down.
func balloonProvider(transactionId uuid.UUID, r Room, occupancy byte, remove bool) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), r.OwnerId(), minigame.EventTypeBalloonUpdated, minigame.BalloonEventBody{
		Remove:      remove,
		RoomType:    r.RoomType(),
		Title:       r.Title(),
		HasPassword: r.Private() && r.Password() != "",
		PieceType:   r.PieceType(),
		Occupancy:   occupancy,
		InProgress:  r.InProgress(),
	})
}
