package trade

import (
	invitemsg "atlas-trades/kafka/message/invite"
	trademsg "atlas-trades/kafka/message/trade"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/invite"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// statusEventProvider builds one EVENT_TOPIC_TRADE_STATUS message. The key is
// the map id so a room's events stay ordered relative to each other, matching
// the mini-game precedent (atlas-mini-games game/producer.go:63-79).
//
// The room identity is passed field by field rather than as a Room because the
// validation ladder rejects some commands BEFORE a room exists — a create that
// fails on the dead check has a field and a character, and nothing else.
func statusEventProvider[E any](txId uuid.UUID, f field.Model, roomId uuid.UUID, handle uint32, roomType byte, ownerId character.Id, visitorId character.Id, characterId character.Id, eventType string, body E) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &trademsg.StatusEvent[E]{
		TransactionId: txId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		RoomId:        roomId,
		Handle:        handle,
		RoomType:      roomType,
		OwnerId:       ownerId,
		VisitorId:     visitorId,
		CharacterId:   characterId,
		Type:          eventType,
		Body:          body,
	}
	return producer.SingleMessageProvider(key, value)
}

// roomEventProvider is statusEventProvider with the room identity unpacked from
// a live Room.
func roomEventProvider[E any](txId uuid.UUID, r Room, characterId character.Id, eventType string, body E) model.Provider[[]kafka.Message] {
	return statusEventProvider(txId, r.Field(), r.Id(), r.Handle(), r.RoomType(), r.OwnerId(), r.VisitorId(), characterId, eventType, body)
}

// errorProvider announces a mini-room enter error by its semantic KEY string,
// for a command that failed before any room existed. atlas-channel resolves the
// key to the per-version numeric code (DOM-25).
func errorProvider(txId uuid.UUID, f field.Model, roomType byte, characterId character.Id, code string) model.Provider[[]kafka.Message] {
	return statusEventProvider(txId, f, uuid.Nil, 0, roomType, characterId, 0, characterId, trademsg.StatusTypeError, trademsg.ErrorEventBody{Code: code})
}

// roomErrorProvider announces a mini-room enter error against an existing room.
func roomErrorProvider(txId uuid.UUID, r Room, characterId character.Id, code string) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, characterId, trademsg.StatusTypeError, trademsg.ErrorEventBody{Code: code})
}

// roomCreatedProvider announces the owner's freshly opened room. Position is
// always 0 — the creator is the owner (FR-1.1, FR-1.5).
func roomCreatedProvider(txId uuid.UUID, r Room) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, r.OwnerId(), trademsg.StatusTypeRoomCreated, trademsg.RoomCreatedEventBody{Position: 0})
}

// inviteSentProvider announces that an invite went out, so the channel can draw
// the target's invite dialog. CharacterId is the INVITER — the acting character
// — while the body names the target.
func inviteSentProvider(txId uuid.UUID, r Room, targetCharacterId character.Id, inviterName string) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, r.OwnerId(), trademsg.StatusTypeInviteSent, trademsg.InviteSentEventBody{
		TargetCharacterId: targetCharacterId,
		InviterName:       inviterName,
	})
}

// inviteRejectedProvider tells the inviter their invite was refused, carrying an
// inviteResult KEY string (see the inviteResult* constants in processor.go).
func inviteRejectedProvider(txId uuid.UUID, r Room, code string) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, r.OwnerId(), trademsg.StatusTypeInviteRejected, trademsg.ErrorEventBody{Code: code})
}

// participantEnteredProvider announces the visitor taking seat 1.
func participantEnteredProvider(txId uuid.UUID, r Room, characterId character.Id, name string) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, characterId, trademsg.StatusTypeParticipantEntered, trademsg.ParticipantEnteredEventBody{
		CharacterId: characterId,
		Name:        name,
		Position:    1,
	})
}

// cancelledProvider tears the room down on both clients, carrying the semantic
// leaveReason KEY string the channel resolves to a per-version status byte.
// characterId is the character whose action triggered the teardown.
func cancelledProvider(txId uuid.UUID, r Room, characterId character.Id, reason string) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, characterId, trademsg.StatusTypeCancelled, trademsg.CancelledEventBody{Reason: reason})
}

// inviteCommandProvider issues the COMMAND_TOPIC_INVITE CREATE that hands the
// offer to atlas-invites. The referenceId is the room's uint32 wire handle:
// invite.Id is a uint32 (libs/atlas-constants/invite/constants.go:3), so the
// room's uuid does not fit — this is why Room carries both ids (design §2.3).
func inviteCommandProvider(txId uuid.UUID, r Room, targetCharacterId character.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(r.Handle()))
	value := &invitemsg.Command[invitemsg.CreateCommandBody]{
		TransactionId: txId,
		WorldId:       r.Field().WorldId(),
		InviteType:    invite.TypeTrade,
		Type:          invite.CommandTypeCreate,
		Body: invitemsg.CreateCommandBody{
			OriginatorId: r.OwnerId(),
			TargetId:     targetCharacterId,
			ReferenceId:  invite.Id(r.Handle()),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
