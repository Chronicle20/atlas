package game

import (
	"atlas-mini-games/kafka/message/minigame"
	"atlas-mini-games/record"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// resultTypeKey maps the internal result-type byte enum (resultWin/resultTie/
// resultForfeit) to the semantic leaveReason/resultType KEY string emitted on
// the wire. The channel resolves the key to a per-version numeric code via the
// tenant resultType table (DOM-25); the internal byte enum is retained for
// game-resolution logic.
func resultTypeKey(resultType byte) string {
	switch resultType {
	case resultWin:
		return "WIN"
	case resultTie:
		return "TIE"
	case resultForfeit:
		return "FORFEIT"
	default:
		return ""
	}
}

// leaveStatusKey maps the internal leave-status byte enum (leaveStatusClosed/
// leaveStatusLeft/leaveStatusExpelled) to the semantic leaveReason KEY string
// emitted on the wire. The channel resolves it to a per-version numeric code
// via the tenant leaveReason table (DOM-25).
func leaveStatusKey(status byte) string {
	switch status {
	case leaveStatusClosed:
		return "MINIGAME_CLOSED"
	case leaveStatusLeft:
		return "MINIGAME_LEFT"
	case leaveStatusExpelled:
		return "MINIGAME_EXPELLED"
	default:
		return ""
	}
}

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
// given leave status (leaveStatusLeft / leaveStatusExpelled). The byte enum is
// mapped to a leaveReason KEY string at this emission boundary (DOM-25).
// characterId is the character that occupied the slot.
func leftProvider(transactionId uuid.UUID, r Room, slot byte, status byte, characterId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), characterId, characterId, minigame.EventTypeLeft, minigame.LeftEventBody{
		Slot:   slot,
		Status: leaveStatusKey(status),
	})
}

// roomClosedProvider announces a ROOM_CLOSED event to the room, carrying the
// visitor's leave status (leaveStatusClosed) as a leaveReason KEY string.
func roomClosedProvider(transactionId uuid.UUID, r Room, visitorStatus byte) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), r.OwnerId(), minigame.EventTypeRoomClosed, minigame.RoomClosedEventBody{
		VisitorStatus: leaveStatusKey(visitorStatus),
	})
}

func chatProvider(transactionId uuid.UUID, r Room, slot byte, characterId uint32, msg string) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), characterId, minigame.EventTypeChat, minigame.ChatEventBody{
		Slot:    slot,
		Message: msg,
	})
}

// readyProvider announces a READY (ready) or UNREADY (!ready) event for the
// room, keyed to the visitor toggling their ready button. Bodyless (§G5).
func readyProvider(transactionId uuid.UUID, r Room, characterId uint32, ready bool) model.Provider[[]kafka.Message] {
	eventType := minigame.EventTypeReady
	if !ready {
		eventType = minigame.EventTypeUnready
	}
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), characterId, eventType, minigame.EmptyEventBody{})
}

// startedProvider announces a STARTED event. FirstMover is the START wire byte
// (the second mover's slot per §G1); the client grants the first move to the
// other slot. deck is the shuffled MatchCards deck (nil/empty for Omok).
func startedProvider(transactionId uuid.UUID, r Room, deck []uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), r.OwnerId(), minigame.EventTypeStarted, minigame.StartedEventBody{
		RoomType:   r.RoomType(),
		FirstMover: r.FirstMover(),
		Deck:       deck,
	})
}

// stonePlacedProvider announces a STONE_PLACED event (Omok). characterId is the
// placing player; StoneType is the placed stone's 1-based color.
func stonePlacedProvider(transactionId uuid.UUID, r Room, x uint32, y uint32, stoneType byte, characterId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), characterId, minigame.EventTypeStonePlaced, minigame.StonePlacedEventBody{
		X:         x,
		Y:         y,
		StoneType: stoneType,
	})
}

// putStoneErrorProvider announces a PUT_STONE_ERROR event (Omok invalid move).
// It is targeted at the acting character only (the mover whose placement was
// rejected); code is a putStoneError KEY (DOUBLE_THREE/CANNOT_PLACE) the channel
// resolves to a per-version numeric byte (DOM-25).
func putStoneErrorProvider(transactionId uuid.UUID, r Room, characterId uint32, code string) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), characterId, minigame.EventTypePutStoneError, minigame.PutStoneErrorEventBody{Code: code})
}

// cardFlippedProvider announces a CARD_FLIPPED event (MatchCards). For a first
// flip (secondFlip=false) the channel forwards it to the opponent only; a second
// flip is broadcast to both (design §3.2). Slot and FirstSlot are card indices;
// ResultType is 0/1 mismatch owner/visitor, 2/3 match owner/visitor.
func cardFlippedProvider(transactionId uuid.UUID, r Room, secondFlip bool, slot byte, firstSlot byte, resultType byte, characterId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), characterId, minigame.EventTypeCardFlipped, minigame.CardFlippedEventBody{
		SecondFlip: secondFlip,
		Slot:       slot,
		FirstSlot:  firstSlot,
		ResultType: resultType,
	})
}

// tieRequestedProvider announces a TIE_REQUESTED event; characterId is the
// requester and the channel targets the opponent (design §3.3).
func tieRequestedProvider(transactionId uuid.UUID, r Room, characterId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), characterId, minigame.EventTypeTieRequested, minigame.EmptyEventBody{})
}

// tieAnsweredProvider announces a TIE_ANSWERED event (decline only — accept
// resolves via GAME_ENDED); characterId is the answerer and the channel targets
// the original requester.
func tieAnsweredProvider(transactionId uuid.UUID, r Room, characterId uint32, accept bool) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), characterId, minigame.EventTypeTieAnswered, minigame.AnswerEventBody{Accept: accept})
}

// retreatRequestedProvider announces a RETREAT_REQUESTED event; characterId is
// the requester and the channel targets the opponent (§G2).
func retreatRequestedProvider(transactionId uuid.UUID, r Room, characterId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), characterId, minigame.EventTypeRetreatRequested, minigame.EmptyEventBody{})
}

// retreatAnsweredProvider announces a RETREAT_ANSWERED event; characterId is the
// answerer. On accept the server already popped the stone (§G2).
func retreatAnsweredProvider(transactionId uuid.UUID, r Room, characterId uint32, accept bool) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), characterId, minigame.EventTypeRetreatAnswered, minigame.AnswerEventBody{Accept: accept})
}

// skippedProvider announces a SKIPPED event. Who is the NEXT-mover slot (== new
// CurrentTurn): owner-skip emits 1, visitor-skip emits 0, matching
// getMiniGameSkipOwner(0x01)/getMiniGameSkipVisitor(0x00) read as next-mover per
// ida-notes §G5. characterId is the skipper.
func skippedProvider(transactionId uuid.UUID, r Room, who byte, characterId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), characterId, minigame.EventTypeSkipped, minigame.SkippedEventBody{Who: who})
}

// gameEndedProvider announces a GAME_ENDED event carrying the resolved result,
// both refreshed persistent records, and the post-game session scores.
func gameEndedProvider(transactionId uuid.UUID, r Room, resultType byte, winnerSlot byte, ownerRecord record.Model, visitorRecord record.Model) model.Provider[[]kafka.Message] {
	return statusEventProvider(transactionId, r.Field(), r.Id(), r.OwnerId(), r.VisitorId(), r.OwnerId(), minigame.EventTypeGameEnded, minigame.GameEndedEventBody{
		ResultType:    resultTypeKey(resultType),
		WinnerSlot:    winnerSlot,
		OwnerRecord:   recordBody(ownerRecord),
		VisitorRecord: recordBody(visitorRecord),
		OwnerScore:    r.OwnerScore(),
		VisitorScore:  r.VisitorScore(),
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
