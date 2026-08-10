package trade

import (
	invitemsg "atlas-trades/kafka/message/invite"
	trademsg "atlas-trades/kafka/message/trade"
	"atlas-trades/settlement"

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
//
// OwnerId and VisitorId are deliberately ZERO: there is no room, so there are no
// participants, and putting the acting character in OwnerId would read
// downstream as a real room owner. CharacterId names who to answer, which is all
// the ERROR handler needs.
func errorProvider(txId uuid.UUID, f field.Model, roomType byte, characterId character.Id, code string) model.Provider[[]kafka.Message] {
	return statusEventProvider(txId, f, uuid.Nil, 0, roomType, 0, 0, characterId, trademsg.StatusTypeError, trademsg.ErrorEventBody{Code: code})
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
// inviteResult KEY string (see the inviteResult* constants in processor.go) and
// the refused target's name, which the client interpolates into every refusal
// message except CANNOT_FIND_CHARACTER's.
func inviteRejectedProvider(txId uuid.UUID, r Room, code string, targetName string) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, r.OwnerId(), trademsg.StatusTypeInviteRejected, trademsg.InviteRejectedEventBody{
		Code:       code,
		TargetName: targetName,
	})
}

// participantEnteredProvider announces the visitor taking seat 1.
func participantEnteredProvider(txId uuid.UUID, r Room, characterId character.Id, name string) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, characterId, trademsg.StatusTypeParticipantEntered, trademsg.ParticipantEnteredEventBody{
		CharacterId: characterId,
		Name:        name,
		Position:    1,
	})
}

// itemStagedProvider announces one staged item. The body names the staging SIDE
// by position rather than by character: atlas-channel converts that to each
// recipient's own recipient-relative side byte before writing the packet, so
// this event is broadcast-shaped and needs no per-recipient variant.
func itemStagedProvider(txId uuid.UUID, r Room, characterId character.Id, position byte, i StagedItem) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, characterId, trademsg.StatusTypeItemStaged, trademsg.ItemStagedEventBody{
		Position:      position,
		TradeSlot:     i.TradeSlot(),
		InventoryType: i.InventoryType(),
		SourceSlot:    i.SourceSlot(),
		AssetId:       i.AssetId(),
		TemplateId:    i.TemplateId(),
		Quantity:      i.Quantity(),
	})
}

// mesoStagedProvider announces the participant's staged meso total. Mode 16 is
// an ASSIGNMENT on the client (design §1.6), so the amount is always the
// absolute staged total and never a delta.
func mesoStagedProvider(txId uuid.UUID, r Room, characterId character.Id, position byte, amount uint32) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, characterId, trademsg.StatusTypeMesoStaged, trademsg.MesoStagedEventBody{
		Position: position,
		Amount:   amount,
	})
}

// mesoRefusedProvider drives the authoritative re-echo (FR-4.8, design §4.2):
// the client already moved its own view to the amount it asked for, and because
// mode 16 assigns rather than accumulates, re-sending the LAST VALID amount is
// what snaps that view back.
// itemRefusedProvider announces a stage that never reached escrow. It is
// addressed to the STAGING character alone: the counterparty was never told the
// item existed, so it has nothing to correct (design §5A.4).
func itemRefusedProvider(txId uuid.UUID, r Room, characterId character.Id, position byte, tradeSlot byte) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, characterId, trademsg.StatusTypeItemRefused, trademsg.ItemRefusedEventBody{
		Position:  position,
		TradeSlot: tradeSlot,
	})
}

func mesoRefusedProvider(txId uuid.UUID, r Room, characterId character.Id, position byte, lastValid uint32) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, characterId, trademsg.StatusTypeMesoRefused, trademsg.MesoRefusedEventBody{
		Position:        position,
		LastValidAmount: lastValid,
	})
}

// participantConfirmedProvider announces that one side pressed Trade. The body
// names the confirming SIDE by position, exactly as ITEM_STAGED does, so
// atlas-channel can convert it to each recipient's recipient-relative side byte.
//
// It is emitted on EVERY confirm, including the first. What is NOT emitted on
// the first confirm is ATTESTATION_REQUESTED (design §6.2): clientbound mode 17
// makes the receiving client auto-reply TRANSACTION, so sending it before both
// sides have confirmed would let one side drive the other's attestation.
func participantConfirmedProvider(txId uuid.UUID, r Room, characterId character.Id, position byte) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, characterId, trademsg.StatusTypeParticipantConfirmed, trademsg.ParticipantConfirmedEventBody{
		Position: position,
	})
}

// attestationRequestedProvider prompts BOTH clients for their CRC attestation
// (clientbound mode 17). CharacterId names the second confirmer — the character
// whose action produced the transition — while the event itself is addressed to
// the room.
func attestationRequestedProvider(txId uuid.UUID, r Room, characterId character.Id) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, characterId, trademsg.StatusTypeAttestationRequested, trademsg.AttestationRequestedEventBody{})
}

// recordEventProvider is roomEventProvider for a room that may no longer
// exist: the room identity comes from the DURABLE settlement record instead.
// The terminal path uses it unconditionally — including when this process does
// still hold the room — so that a settlement completed after a restart is
// byte-identical to one completed live.
//
// CharacterId is the owner. Unlike a cancel, a terminal settlement has no
// triggering character: the FAILED event names the failed expanded step's
// character, which is not a role and must never be read as one.
func recordEventProvider[E any](txId uuid.UUID, s settlement.Model, eventType string, body E) model.Provider[[]kafka.Message] {
	return statusEventProvider(txId, s.Field(), s.RoomId(), s.Handle(), s.RoomType(), s.OwnerId(), s.VisitorId(), s.OwnerId(), eventType, body)
}

// recordSettledProvider announces a completed trade, which atlas-channel writes
// as LEAVE 7. Per design §6.4 it is emitted ONLY after the settlement saga
// reports terminal success, because the client renders its "received %d mesos
// after fees" line from its own character data.
func recordSettledProvider(txId uuid.UUID, s settlement.Model, ledgerEntryId uuid.UUID) model.Provider[[]kafka.Message] {
	return recordEventProvider(txId, s, trademsg.StatusTypeSettled, trademsg.SettledEventBody{
		LedgerEntryId: ledgerEntryId,
	})
}

// recordCancelledProvider tears both dialogs down after a settlement failed,
// carrying the leaveReason KEY atlas-channel resolves to a status byte.
func recordCancelledProvider(txId uuid.UUID, s settlement.Model, reason string) model.Provider[[]kafka.Message] {
	return recordEventProvider(txId, s, trademsg.StatusTypeCancelled, trademsg.CancelledEventBody{Reason: reason})
}

// cancelledProvider tears the room down on both clients, carrying the semantic
// leaveReason KEY string the channel resolves to a per-version status byte.
// characterId is the character whose action triggered the teardown.
func cancelledProvider(txId uuid.UUID, r Room, characterId character.Id, reason string) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, characterId, trademsg.StatusTypeCancelled, trademsg.CancelledEventBody{Reason: reason})
}

// chatProvider relays one room chat line. The event addresses the whole room
// (OwnerId and VisitorId both ride the envelope), and Body.Position names which
// side spoke so atlas-channel can render the speaker's name prefix; CharacterId
// is the speaker.
func chatProvider(txId uuid.UUID, r Room, characterId character.Id, position byte, message string) model.Provider[[]kafka.Message] {
	return roomEventProvider(txId, r, characterId, trademsg.StatusTypeChat, trademsg.ChatEventBody{Position: position, Message: message})
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

// inviteRejectCommandProvider retires the offer in atlas-invites when the client
// declined it directly to us. Leaving it live is not merely untidy: a room's
// handle defaults to the owner's character id, so the owner's NEXT room reuses
// the same referenceId, and the invite registry's dedup
// (services/atlas-invites/atlas.com/invites/invite/registry.go:89-107 returns
// the existing invite on a referenceId match) would hand back the stale invite
// instead of raising a fresh dialog until the old one timed out.
func inviteRejectCommandProvider(txId uuid.UUID, r Room, targetCharacterId character.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(r.Handle()))
	value := &invitemsg.Command[invitemsg.RejectCommandBody]{
		TransactionId: txId,
		WorldId:       r.Field().WorldId(),
		InviteType:    invite.TypeTrade,
		Type:          invite.CommandTypeReject,
		Body: invitemsg.RejectCommandBody{
			TargetId:     targetCharacterId,
			OriginatorId: r.OwnerId(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
