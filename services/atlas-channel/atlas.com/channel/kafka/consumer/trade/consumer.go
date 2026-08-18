// Package trade turns atlas-trades' EVENT_TOPIC_TRADE_STATUS events back into
// CharacterInteraction packets. atlas-channel decodes the wire and writes
// packets; it never mutates inventory or meso for a trade — all trade state
// lives in atlas-trades (task-205 design §2.2).
package trade

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	trade2 "atlas-channel/kafka/message/trade"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	socketmodel "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	tradeproc "atlas-channel/trade"
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	atlasmodel "github.com/Chronicle20/atlas/libs/atlas-model/model"
	atlaspacket "github.com/Chronicle20/atlas/libs/atlas-packet"
	interactionpkt "github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	interactioncb "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	// ownerPosition and visitorPosition are the two seats of a trade room. They
	// are ROOM positions, not wire codes: CMiniRoomBaseDlg::OnEnterResultBase
	// reads the recipient's own seat as the second header byte and keys its
	// visitor list on the same numbering, and atlas-trades stamps the same two
	// values into every status body's Position field
	// (services/atlas-trades/.../trade/producer.go — RoomCreated 0, Entered 1).
	ownerPosition   = byte(0)
	visitorPosition = byte(1)

	// ownSide and counterpartySide are the two values of the RECIPIENT-RELATIVE
	// side byte the trade dialog reads in its PUT_ITEM and ADD_MESO arms
	// (CTradingRoomDlg::OnPutItem @0x7c1fb7, ::OnPutMoney @0x7c208e): the client
	// indexes its own grid with 0 and the counterparty's with 1. They are a
	// side-of-the-dialog selector the receiving client resolves locally, not a
	// per-version code, so they are not tenant-resolved — unlike every mode
	// byte, enter error and leave status this package emits, which are.
	ownSide          = byte(0)
	counterpartySide = byte(1)
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...atlasmodel.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...atlasmodel.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("trade_status_event")(trade2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var handles []listener.HandlerHandle
				t, _ := topic.EnvProvider(l)(trade2.EnvEventTopicStatus)()
				for _, h := range []handler.Handler{
					message.AdaptHandler(message.PersistentConfig(handleRoomCreatedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleParticipantEnteredEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleInviteSentEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleInviteRejectedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleItemStagedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleItemRefusedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleMesoStagedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleMesoRefusedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleParticipantConfirmedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleAttestationRequestedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleSettledEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleCancelledEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleErrorEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleChatEvent(sc, wp))),
				} {
					id, err := rf(t, h)
					if err != nil {
						return nil, err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				}
				return handles, nil
			}
		}
	}
}

// guard applies the tenant/world/channel ownership check every handler runs
// before acting on an event. EVENT_TOPIC_TRADE_STATUS fans out to every channel
// in the world, so an event for another channel must be dropped silently.
func guard[E any](sc server.Model, ctx context.Context, e trade2.StatusEvent[E]) bool {
	return sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId)
}

// positionOf resolves a character's ABSOLUTE position in the room from the
// event's owner/visitor ids.
func positionOf[E any](e trade2.StatusEvent[E], characterId charconst.Id) byte {
	if characterId == e.VisitorId {
		return visitorPosition
	}
	return ownerPosition
}

// sideFor converts the event's ABSOLUTE room position (0 owner, 1 visitor) into
// the RECIPIENT-RELATIVE side byte the client reads: 0 means "my own side of
// the dialog", 1 means "the counterparty's". Sending one byte to both clients
// puts the item on the wrong side of one of the two windows.
func sideFor(stagerPosition byte, recipientPosition byte) byte {
	if stagerPosition == recipientPosition {
		return ownSide
	}
	return counterpartySide
}

// roomOccupants lists the room's occupants in seat order. An event raised
// before a room existed carries zero ids (atlas-trades' errorProvider), and a
// room whose invite has not been accepted has no visitor yet, so both are
// skipped rather than announced to character 0.
func roomOccupants[E any](e trade2.StatusEvent[E]) []charconst.Id {
	var ids []charconst.Id
	if e.OwnerId != 0 {
		ids = append(ids, e.OwnerId)
	}
	if e.VisitorId != 0 && e.VisitorId != e.OwnerId {
		ids = append(ids, e.VisitorId)
	}
	return ids
}

// announceTo sends one body to a single character, dropping the write when
// there is no such character. Zero is not a character id: an event raised
// before a room existed carries zero participant ids (atlas-trades'
// errorProvider), and a body addressed to id 0 would be a session lookup that
// can only miss. The check lives HERE rather than inside the seam so it is
// production code the tests exercise, not stub behaviour they reimplement.
func announceTo(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId charconst.Id, body packet.Encode) {
	if characterId == 0 {
		return
	}
	tradeAnnouncer(l, ctx, sc, wp, characterId, body)
}

// tradeAnnouncer is the channel-side seam that resolves a character's session
// and announces one CharacterInteraction body to it. Package-level var so tests
// can swap in a recording stub without a live net.Conn or a real writer
// registry (mirrors the RPS consumer's rpsAnnouncer seam).
var tradeAnnouncer = func(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId charconst.Id, body packet.Encode) {
	err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(characterId),
		session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(body))
	if err != nil {
		l.WithError(err).Errorf("Unable to announce CharacterInteraction frame to character [%d].", characterId)
	}
}

// unlockClient releases the acting client's exclusive-request lock.
//
// The v83 client arms CWvsContext::m_bExclRequestSent when it sends PUT_ITEM
// (CTradingRoomDlg::PutItem @0x7c359f) or PUT_MONEY (::PutMoney @0x7c37ca), and
// CanSendExclRequest @0x485bf7 then refuses BOTH until a server packet clears
// it. A stage that succeeds clears it for free — the inventory or meso mutation
// produces a delta whose leading exclRequestSent bool does the job. A stage that
// is REFUSED produces no mutation, so without this the trade dialog is wedged
// for the rest of the session: the mesos button and every further put-item stop
// responding, which is exactly the defect the escrow amendment was written to
// fix (design §5A.6).
//
// The slot stays empty either way. This adds no visible feedback; it only
// releases the lock.
func unlockClient(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId charconst.Id) {
	if characterId == 0 {
		return
	}
	tradeUnlocker(l, ctx, sc, wp, characterId)
}

// tradeUnlocker is the seam that resolves a character's session and sends the
// bare unlock. Package-level var for the same reason tradeAnnouncer is one.
var tradeUnlocker = func(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId charconst.Id) {
	err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(characterId),
		session.EnableActions(l)(ctx)(wp))
	if err != nil {
		l.WithError(err).Errorf("Unable to release the action lock for character [%d].", characterId)
	}
}

// tradeMesoLimitConfigured reports whether this tenant's CharacterInteraction
// writer binds the TRADE_MESO_LIMIT operation, i.e. whether the client version
// has that dispatcher arm at all. Package-level var so tests can exercise both
// the present and absent versions without standing up a writer registry.
var tradeMesoLimitConfigured = func(l logrus.FieldLogger, ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	opts, ok := writer.TenantWriterOptions(t.Id(), interactioncb.CharacterInteractionWriter)
	if !ok {
		l.Warnf("Writer options for [%s] missing; TRADE_MESO_LIMIT not sent.", interactioncb.CharacterInteractionWriter)
		return false
	}
	return atlaspacket.CodeConfigured(opts, "operations", interactioncb.CharacterInteractionModeTradeMesoLimit)
}

// tradeEnterErrorConfigured reports whether this tenant's CharacterInteraction
// writer binds the given `enterError` KEY, i.e. whether the client version has
// a mini-room enter-error arm for that refusal at all.
//
// The ENTER_RESULT mode itself is bound on every version, so the frame is
// always dispatchable; it is the trailing REASON byte that is version-specific.
// ResolveCode falls back to 99 on a miss — self-documented as "will likely
// cause a client crash" — and the shipped tables are not uniform: gms_v92 and
// gms_v12 bind no `enterError` table at all, and jms_185 binds only
// TRADE_NOT_ALLOWED / TRADE_NOT_ALLOWED_2 (task-23 derived those two per-IDB
// and correctly refused to guess the rest, because the JMS switch is
// demonstrably non-positional). Every atlas-trades refusal key would therefore
// write 99 on those tenants.
//
// Config stays the per-version authority (DOM-25): no version is named here,
// and no table is back-filled by copying v83's — that inherited-copy habit is
// what produced the gms_v48 +1 serverbound bug this branch fixed. A refusal
// with no bound code is dropped, exactly as TRADE_MESO_LIMIT is; the player
// sees no message rather than a crashed client.
//
// Package-level var so tests can exercise both the present and absent versions
// without standing up a writer registry.
var tradeEnterErrorConfigured = func(l logrus.FieldLogger, ctx context.Context, key interactioncb.CharacterInteractionEnterErrorMode) bool {
	t := tenant.MustFromContext(ctx)
	opts, ok := writer.TenantWriterOptions(t.Id(), interactioncb.CharacterInteractionWriter)
	if !ok {
		l.Warnf("Writer options for [%s] missing; enter error [%s] not sent.", interactioncb.CharacterInteractionWriter, key)
		return false
	}
	return atlaspacket.CodeConfigured(opts, "enterError", key)
}

// announceToRoom sends the SAME body to every occupant. Bodies that carry a
// recipient-relative field (the staged side byte, the leave slot) must not use
// this — they build a per-recipient body instead.
func announceToRoom[E any](l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, e trade2.StatusEvent[E], body packet.Encode) {
	for _, id := range roomOccupants(e) {
		announceTo(l, ctx, sc, wp, id, body)
	}
}

// tradeCancelRequester is the seam that asks atlas-trades to tear a room down.
// CANCEL is the only teardown verb in the trade command contract, it is
// idempotent (TeardownCharacter is a no-op for a character with no room —
// kafka/consumer/trade/consumer.go in atlas-trades), and it releases every
// staged item's reservation, so it is safe to send from a handler that may be
// racing the room's own teardown. Package-level var so tests can record the
// request without a broker.
var tradeCancelRequester = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId charconst.Id) error {
	return tradeproc.NewProcessor(l, ctx).Cancel(f, characterId)
}

// abortOnDisplayFailure tears the room down when atlas-channel cannot render a
// piece of state that BOTH clients must see before they can consent.
//
// The trade dialog is the whole consent mechanism. atlas-trades' attestation
// compares each client's confirm list to ITS OWN attest list
// (settlement.go entriesMatch), never to the server's staged list, so a frame
// that never reached either client cannot be caught at settle: the item is
// escrowed and transferred while neither player ever saw it. A REST hiccup
// resolving a participant's avatar must therefore not degrade to "settle
// anyway" — it must stop the trade.
//
// Its remaining callers are all avatar resolution: the room-created and
// participant-entered enter-result frames. The staged-item frame no longer needs
// it, because the item block now travels on the event instead of being read back
// (see handleItemStagedEvent) and there is nothing left there that can fail.
//
// Cancelling is safe for the participant who has no dialog open yet (the
// room-created / participant-entered failures): CMiniRoomBaseDlg::OnPacketBase
// (v83 @0x65df4c) dispatches mode 10 LEAVE only through the live-dialog branch
// and returns early when TSingleton<CUniqueModeless>::ms_pInstance is null, so
// the resulting LEAVE is a no-op there rather than the CDisconnectException an
// OnLeaveBase against an unpopulated avatar slot would raise.
//
// A retry alone would not be sufficient: a retry that still fails leaves the
// same invisible-transfer window open.
func abortOnDisplayFailure[E any](l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[E], characterId charconst.Id, cause error, what string) {
	l.WithError(cause).Errorf("Unable to render %s for trade room [%s]; cancelling the trade rather than settling state neither client can see.", what, e.RoomId)
	if characterId == 0 {
		return
	}
	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
	if err := tradeCancelRequester(l, ctx, f, characterId); err != nil {
		l.WithError(err).Errorf("Unable to cancel trade room [%s] after a display failure. The room may remain open until the participant changes map, channel or logs out.", e.RoomId)
	}
}

// tradeRoomVisitorResolver is the seam that resolves a character into the
// {slot, avatar, name} visitor entry the enter-result frame carries. Package-
// level var so the enter-result shape can be tested without a REST round trip —
// that shape is load-bearing: CMiniRoomBaseDlg::OnEnterResultBase (@0x65ec3d)
// populates the dialog's avatar array EXCLUSIVELY from this list, and
// ::OnLeaveBase (@0x65edb5) throws CDisconnectException on a LEAVE naming a slot
// the array never filled. An omitted entry is a client disconnect, not a
// cosmetic gap.
var tradeRoomVisitorResolver = tradeRoomVisitor

// tradeRoomVisitor resolves a character into the {slot, avatar, name} visitor
// entry the enter-result frame carries.
func tradeRoomVisitor(l logrus.FieldLogger, ctx context.Context, slot byte, characterId charconst.Id) (interactionpkt.Visitor, error) {
	cp := character.NewProcessor(l, ctx)
	c, err := cp.GetById(cp.InventoryDecorator)(uint32(characterId))
	if err != nil {
		return interactionpkt.Visitor{}, err
	}
	return interactionpkt.NewBaseVisitor(slot, socketmodel.NewFromCharacter(c, false), c.Name()), nil
}

// handleRoomCreatedEvent opens the creator's own trade dialog. The frame is the
// base enter-result frame (design §1.3 — CTradingRoomDlg's enter-result tail
// virtual is nullsub_94, so nothing follows), and it carries the owner's OWN
// {slot 0, avatar, name} entry: CMiniRoomBaseDlg::OnEnterResultBase builds the
// dialog's avatar list from that 0xFF-terminated list, so an empty list opens a
// window with the creator's own side unpopulated.
func handleRoomCreatedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.RoomCreatedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.RoomCreatedEventBody]) {
		if e.Type != trade2.StatusTypeRoomCreated {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Trade room [%s] created by character [%d]. roomType [%d], position [%d].", e.RoomId, e.OwnerId, e.RoomType, e.Body.Position)
		owner, err := tradeRoomVisitorResolver(l, ctx, e.Body.Position, e.OwnerId)
		if err != nil {
			// No dialog was drawn, but atlas-trades has already seated the owner:
			// they can neither create another room (ErrOwnerHasRoom) nor enter a
			// mini-game, and they have no dialog to close, so CANCEL never comes.
			abortOnDisplayFailure(l, ctx, e, e.OwnerId, err, fmt.Sprintf("owner [%d]'s enter-result", e.OwnerId))
			return
		}
		room := interactionpkt.NewTradeRoom(interactionpkt.RoomType(e.RoomType), e.Body.Position, []interactionpkt.Visitor{owner})
		announceTo(l, ctx, sc, wp, e.OwnerId, interactioncb.CharacterInteractionEnterResultSuccessBody(room))
	}
}

// handleParticipantEnteredEvent opens the entrant's dialog with BOTH occupants
// (its own seat named by the frame's position byte) and tells the owner a
// visitor arrived, which appends the new avatar to the already-open dialog.
func handleParticipantEnteredEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.ParticipantEnteredEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.ParticipantEnteredEventBody]) {
		if e.Type != trade2.StatusTypeParticipantEntered {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Character [%d] entered trade room [%s] at position [%d].", e.Body.CharacterId, e.RoomId, e.Body.Position)
		owner, err := tradeRoomVisitorResolver(l, ctx, ownerPosition, e.OwnerId)
		if err != nil {
			// The entrant is seated in atlas-trades but gets no dialog, and the
			// owner's already-open dialog never learns of them — a room neither
			// side can act on. Tear it down instead of leaving it soft-locked.
			abortOnDisplayFailure(l, ctx, e, e.Body.CharacterId, err, fmt.Sprintf("owner [%d]'s entry in the enter-result", e.OwnerId))
			return
		}
		visitor, err := tradeRoomVisitorResolver(l, ctx, e.Body.Position, e.Body.CharacterId)
		if err != nil {
			abortOnDisplayFailure(l, ctx, e, e.Body.CharacterId, err, fmt.Sprintf("visitor [%d]'s entry in the enter-result", e.Body.CharacterId))
			return
		}
		room := interactionpkt.NewTradeRoom(interactionpkt.RoomType(e.RoomType), e.Body.Position, []interactionpkt.Visitor{owner, visitor})
		announceTo(l, ctx, sc, wp, e.Body.CharacterId, interactioncb.CharacterInteractionEnterResultSuccessBody(room))
		announceTo(l, ctx, sc, wp, e.OwnerId, interactioncb.CharacterInteractionEnterBody(visitor))
	}
}

// handleInviteSentEvent draws the target's invite dialog. dwSN is the room's
// uint32 wire handle, which the client echoes back on accept or decline —
// atlas-trades resolves the room from it (design §2.3).
func handleInviteSentEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.InviteSentEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.InviteSentEventBody]) {
		if e.Type != trade2.StatusTypeInviteSent {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Trade invite from [%s] to character [%d]. handle [%d].", e.Body.InviterName, e.Body.TargetCharacterId, e.Handle)
		announceTo(l, ctx, sc, wp, e.Body.TargetCharacterId, interactioncb.CharacterInteractionInviteBody(e.RoomType, e.Body.InviterName, e.Handle))
	}
}

// handleInviteRejectedEvent tells the inviter their offer was refused. The body
// carries an inviteResult KEY string, resolved to the per-version numeric code
// by the tenant inviteResult table inside the body func (DOM-25), and the
// refused target's name. The name is passed straight through: the v83
// CANNOT_FIND_CHARACTER arm reads none (atlas-trades sends "" there), while
// BUSY and the other refusals interpolate it into their message, so dropping it
// would render "%s is doing something else right now" with a blank subject.
func handleInviteRejectedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.InviteRejectedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.InviteRejectedEventBody]) {
		if e.Type != trade2.StatusTypeInviteRejected {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Trade invite rejected for room [%s]. code [%s], targetName [%s].", e.RoomId, e.Body.Code, e.Body.TargetName)
		announceTo(l, ctx, sc, wp, e.CharacterId, interactioncb.CharacterInteractionInviteResultKeyBody(e.Body.Code, e.Body.TargetName))
	}
}

// handleItemStagedEvent announces one staged item to BOTH occupants, each with
// its own recipient-relative side byte. The item block is built once per event
// and reused for both frames.
//
// It renders the snapshot the event carries and performs NO lookup. Under
// escrow-at-staging the asset has already left its owner's compartment for
// atlas-trades' escrow row by the time this fires, so it is in nobody's
// compartment: the read-back this handler used to do could only ever miss, and
// the miss cancelled the trade on the first staged item. Rendering from the
// snapshot removes a REST round trip AND the abortOnDisplayFailure branch that
// hung off it — there is no longer a failure mode here that can leave the item
// invisible to one client, because the frame is built from data that arrived
// with the event.
//
// zeroPosition (inside NewAssetFromSnapshot) is what makes this a bare
// GW_ItemSlotBase with no leading inventory position, the shape
// InteractionTradePutItem.Encode expects and the same one the shop and MTS views
// use. Snapshot.Quantity is the STAGED amount, so a partial stack renders the
// staged count rather than the source stack's.
func handleItemStagedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.ItemStagedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.ItemStagedEventBody]) {
		if e.Type != trade2.StatusTypeItemStaged {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Character [%d] staged item [%d] in trade room [%s]. position [%d], tradeSlot [%d].", e.CharacterId, e.Body.Snapshot.TemplateId, e.RoomId, e.Body.Position, e.Body.TradeSlot)
		a := socketmodel.NewAssetFromSnapshot(e.Body.Snapshot)
		for _, id := range roomOccupants(e) {
			side := sideFor(e.Body.Position, positionOf(e, id))
			announceTo(l, ctx, sc, wp, id, interactioncb.CharacterInteractionTradePutItemBody(side, e.Body.TradeSlot, a))
		}
	}
}

// handleMesoStagedEvent announces a side's ABSOLUTE staged meso total to BOTH
// occupants, each with its own recipient-relative side byte. Mode 16 assigns
// rather than accumulates (design §1.6), so the amount is always the total.
func handleMesoStagedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.MesoStagedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.MesoStagedEventBody]) {
		if e.Type != trade2.StatusTypeMesoStaged {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Character [%d] staged [%d] meso in trade room [%s]. position [%d].", e.CharacterId, e.Body.Amount, e.RoomId, e.Body.Position)
		for _, id := range roomOccupants(e) {
			side := sideFor(e.Body.Position, positionOf(e, id))
			announceTo(l, ctx, sc, wp, id, interactioncb.CharacterInteractionTradeAddMesoBody(side, e.Body.Amount))
		}
	}
}

// handleMesoRefusedEvent corrects the refused client only. The authoritative
// TRADE_ADD_MESO re-echo is what actually snaps the client's view back, because
// mode 16 is an ASSIGNMENT; TRADE_MESO_LIMIT only supplies the reason and is
// absent from the cash trade room on every version (design §4.2). The side byte
// is the refused character's OWN side, since the frame goes to that client
// alone.
func handleMesoRefusedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.MesoRefusedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.MesoRefusedEventBody]) {
		if e.Type != trade2.StatusTypeMesoRefused {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Meso stage refused for character [%d] in trade room [%s]. lastValidAmount [%d].", e.CharacterId, e.RoomId, e.Body.LastValidAmount)
		announceTo(l, ctx, sc, wp, e.CharacterId, interactioncb.CharacterInteractionTradeAddMesoBody(ownSide, e.Body.LastValidAmount))
		// The re-echo above is what actually snaps the client back, so the
		// reason frame is skippable. TRADE_MESO_LIMIT is version-ABSENT on
		// jms_185 — CTradingRoomDlg's dispatcher there has exactly three arms
		// (13/14/15) and no meso-limit case, which is why the matrix grades
		// that cell n-a and template_jms_185_1.json binds no such operation.
		// Announcing regardless would resolve to ResolveCode's 99 sentinel and
		// hand the client a mode it cannot dispatch. Presence in the tenant's
		// operations table is the per-version authority (DOM-25).
		if tradeMesoLimitConfigured(l, ctx) {
			announceTo(l, ctx, sc, wp, e.CharacterId, interactioncb.CharacterInteractionTradeMesoLimitBody())
		}
		// The re-echo and the reason frame are both CharacterInteraction bodies;
		// neither carries the exclRequestSent bool, so neither releases the lock
		// PutMoney armed on send. Without this the refused client can never
		// stage again (design §5A.6).
		unlockClient(l, ctx, sc, wp, e.CharacterId)
	}
}

// handleItemRefusedEvent writes NOTHING BUT THE UNLOCK, and that is the entire
// point of the event.
//
// A refused stage is player-visibly silent by design: the trade slot simply
// stays empty, which is the feedback (design §7). But the client armed
// CWvsContext::m_bExclRequestSent when it sent PUT_ITEM, and
// CWvsContext::CanSendExclRequest then refuses every later exclusive request —
// including ADD_MESO, which is why the reported symptom was "the mesos button
// stopped working after I put an item in". Only the leading exclRequestSent bool
// of STAT_CHANGED or INVENTORY_OPERATION, or a SET_FIELD, clears it, and a
// refused stage produces none of the three: nothing left the inventory.
//
// So there is no trade packet here at all. The empty STAT_CHANGED is the whole
// response (design §5A.6).
func handleItemRefusedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.ItemRefusedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.ItemRefusedEventBody]) {
		if e.Type != trade2.StatusTypeItemRefused {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Item stage refused for character [%d] in trade room [%s], trade slot [%d]. Releasing the action lock.", e.CharacterId, e.RoomId, e.Body.TradeSlot)
		unlockClient(l, ctx, sc, wp, e.CharacterId)
	}
}

// handleParticipantConfirmedEvent writes NO packet. CTradingRoomDlg renders the
// confirming side's state locally when its own user presses Trade, and the
// counterparty learns of it only through the mode-17 attestation prompt, which
// is broadcast once BOTH sides have confirmed (design §6.2). Emitting mode 17
// here would drive the counterparty's attestation without its owner acting.
func handleParticipantConfirmedEvent(sc server.Model, _ writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.ParticipantConfirmedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.ParticipantConfirmedEventBody]) {
		if e.Type != trade2.StatusTypeParticipantConfirmed {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Character [%d] confirmed trade room [%s] at position [%d].", e.CharacterId, e.RoomId, e.Body.Position)
	}
}

// handleAttestationRequestedEvent prompts BOTH clients for their CRC
// attestation at once. Receipt auto-replies with serverbound TRANSACTION, which
// is why atlas-trades emits this only after the second confirm.
func handleAttestationRequestedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.AttestationRequestedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.AttestationRequestedEventBody]) {
		if e.Type != trade2.StatusTypeAttestationRequested {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Attestation requested for trade room [%s].", e.RoomId)
		announceToRoom(l, ctx, sc, wp, e, interactioncb.CharacterInteractionTradeConfirmBody())
	}
}

// handleSettledEvent closes both dialogs with the success leave status. Trade
// completion is not a distinct mode: it is LEAVE + slot + status (design §1.4),
// and the status resolves from the tenant leaveReason table by KEY. Each
// recipient gets its OWN absolute room position in the slot byte, the seat
// CMiniRoomBaseDlg::OnLeaveBase removes.
func handleSettledEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.SettledEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.SettledEventBody]) {
		if e.Type != trade2.StatusTypeSettled {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Trade room [%s] settled. ledgerEntryId [%s].", e.RoomId, e.Body.LedgerEntryId)
		for _, id := range roomOccupants(e) {
			announceTo(l, ctx, sc, wp, id, interactioncb.CharacterInteractionLeaveReasonBody(positionOf(e, id), interactioncb.CharacterInteractionLeaveReasonTradeSuccess))
		}
	}
}

// handleCancelledEvent tears both dialogs down, passing the event's semantic
// leaveReason KEY straight through: the channel resolves it via the tenant
// leaveReason table and never invents a numeric status (DOM-25).
func handleCancelledEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.CancelledEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.CancelledEventBody]) {
		if e.Type != trade2.StatusTypeCancelled {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Trade room [%s] cancelled. reason [%s].", e.RoomId, e.Body.Reason)
		for _, id := range roomOccupants(e) {
			announceTo(l, ctx, sc, wp, id, interactioncb.CharacterInteractionLeaveReasonBody(positionOf(e, id), e.Body.Reason))
		}
	}
}

// handleErrorEvent answers the acting character with a mini-room enter error.
// The body carries the enterError KEY string, resolved to the per-version
// numeric code by the tenant enterError table inside the body func. An error
// raised before a room existed carries zero OwnerId/VisitorId, so this handler
// addresses CharacterId and never the room.
func handleErrorEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.ErrorEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.ErrorEventBody]) {
		if e.Type != trade2.StatusTypeError {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Trade error for character [%d]. code [%s].", e.CharacterId, e.Body.Code)
		if !tradeEnterErrorConfigured(l, ctx, e.Body.Code) {
			l.Infof("Enter error [%s] is not bound in this tenant's enterError table; refusal not sent to character [%d].", e.Body.Code, e.CharacterId)
			return
		}
		announceTo(l, ctx, sc, wp, e.CharacterId, interactioncb.CharacterInteractionEnterResultErrorBody(e.Body.Code))
	}
}

// handleChatEvent relays one room chat line to both occupants. The miniroom
// chat wire carries no separate name field — the client renders the string
// verbatim and splits on " : " only to recolor (CMiniRoomBaseDlg::OnChat, v95
// @0x639AD0) — so the speaker's name is prepended here, matching the mini-game
// and merchant chat paths.
func handleChatEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.ChatEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e trade2.StatusEvent[trade2.ChatEventBody]) {
		if e.Type != trade2.StatusTypeChat {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Chat in trade room [%s] from position [%d].", e.RoomId, e.Body.Position)
		chatText := e.Body.Message
		if name := resolveChatName(l, ctx, e.CharacterId); name != "" {
			chatText = name + " : " + e.Body.Message
		}
		announceToRoom(l, ctx, sc, wp, e, interactioncb.CharacterInteractionChatBody(e.Body.Position, chatText))
	}
}

// resolveChatName looks up the speaking character's name to prepend to a trade
// chat line, returning "" (chat still delivered, just without the prefix) if
// the lookup fails.
func resolveChatName(l logrus.FieldLogger, ctx context.Context, characterId charconst.Id) string {
	c, err := character.NewProcessor(l, ctx).GetById()(uint32(characterId))
	if err != nil {
		l.WithError(err).Warnf("Unable to resolve trade chat sender name for character [%d].", characterId)
		return ""
	}
	return c.Name()
}
