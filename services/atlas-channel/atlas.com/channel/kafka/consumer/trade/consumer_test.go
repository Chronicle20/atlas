package trade

import (
	trade2 "atlas-channel/kafka/message/trade"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	atlaspacket "github.com/Chronicle20/atlas/libs/atlas-packet"
	interactionpkt "github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	interactioncb "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// testOperations is the CharacterInteraction writer "operations" table the test
// encodes against. The values are the gms_v83 mode bytes documented on the
// CharacterInteractionMode* keys; the point of the table is that the consumer
// reads its bytes from HERE rather than from a Go literal, so a wrong key in
// the consumer shows up as a wrong mode byte in the captured frame.
var testOperations = map[string]interface{}{
	"INVITE":           float64(2),
	"INVITE_RESULT":    float64(3),
	"ENTER":            float64(4),
	"ENTER_RESULT":     float64(5),
	"CHAT":             float64(6),
	"CHAT_THING":       float64(8),
	"LEAVE":            float64(10),
	"TRADE_PUT_ITEM":   float64(15),
	"TRADE_ADD_MESO":   float64(16),
	"TRADE_CONFIRM":    float64(17),
	"TRADE_MESO_LIMIT": float64(21),
}

// testLeaveReason is the leaveReason table for the six trade statuses, at the
// v83 bytes CTradingRoomDlg::OnLeave branches on (design §1.4).
var testLeaveReason = map[string]interface{}{
	"TRADE_CANCELLED":     float64(2),
	"TRADE_SUCCESS":       float64(7),
	"TRADE_FAILED":        float64(8),
	"TRADE_CANNOT_CARRY":  float64(9),
	"TRADE_DIFFERENT_MAP": float64(12),
	"TRADE_CRC_FAILED":    float64(13),
}

// testEnterError is the enterError table for the six refusal keys atlas-trades
// emits (services/atlas-trades/.../trade/processor.go), at the gms_v83 bytes.
// It is deliberately a SUBSET of the 22-key shipped table: the point is that
// the consumer looks each key up rather than assuming one exists.
var testEnterError = map[string]interface{}{
	"ROOM_CLOSED":       float64(1),
	"OTHER_REQUESTS":    float64(3),
	"NOT_WHEN_DEAD":     float64(4),
	"UNABLE":            float64(6),
	"TRADE_NOT_ALLOWED": float64(7),
	"NOT_SAME_MAP":      float64(9),
}

// testOptions is the full writer option set a captured body is encoded with.
var testOptions = map[string]interface{}{
	"operations":  testOperations,
	"leaveReason": testLeaveReason,
	"enterError":  testEnterError,
}

// announceCall captures one invocation of the tradeAnnouncer seam: which
// character's session it targeted, and the wire bytes the selected body func
// produced against testOptions.
type announceCall struct {
	characterId charconst.Id
	bytes       []byte
}

// eventDispatch is one prepared status event bound to the handler that owns it.
// The server model is supplied by handleAndCapture, so a test case names only
// the event.
type eventDispatch func(sc server.Model) func(l logrus.FieldLogger, ctx context.Context)

// Named argument types, so a call site reads as
// itemStagedEvent(ownerId(100), visitorId(200), position(0)) rather than as
// three bare numbers.
type (
	ownerId     charconst.Id
	visitorId   charconst.Id
	characterId charconst.Id
	position    byte
	lastValid   uint32
)

const (
	// testChannelId is the channel the test server model owns; every event that
	// should be handled carries it.
	testChannelId = 1
	// testTradeSlot is an arbitrary in-range (1..9) trade grid slot.
	testTradeSlot = byte(3)
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channelconst.NewModel(0, testChannelId)
	return server.NewProcessor(logrus.New(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

// handleAndCapture runs one prepared event through its handler with the
// tradeAnnouncer seam swapped for a recorder, and returns every announce the
// handler made.
func handleAndCapture(t *testing.T, d eventDispatch) []announceCall {
	t.Helper()
	return handleAndCaptureWith(t, d, testOptions)
}

// handleAndCaptureWith is handleAndCapture with an explicit writer option set,
// so a test can model a client version whose operations table binds a
// different set of arms. Both the encode step and the consumer's own
// arm-presence gate read from the SAME table, which is what makes a
// version-absent arm testable without a live writer registry.
func handleAndCaptureWith(t *testing.T, d eventDispatch, opts map[string]interface{}) []announceCall {
	t.Helper()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	l, _ := testlog.NewNullLogger()
	var recorded []announceCall
	cancelled = nil
	orig := tradeAnnouncer
	tradeAnnouncer = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, _ writer.Producer, cid charconst.Id, body packet.Encode) {
		recorded = append(recorded, announceCall{characterId: cid, bytes: body(l, ctx)(opts)})
	}
	defer func() { tradeAnnouncer = orig }()

	origGate := tradeMesoLimitConfigured
	tradeMesoLimitConfigured = func(_ logrus.FieldLogger, _ context.Context) bool {
		return atlaspacket.CodeConfigured(opts, "operations", interactioncb.CharacterInteractionModeTradeMesoLimit)
	}
	defer func() { tradeMesoLimitConfigured = origGate }()

	origEnterGate := tradeEnterErrorConfigured
	tradeEnterErrorConfigured = func(_ logrus.FieldLogger, _ context.Context, key interactioncb.CharacterInteractionEnterErrorMode) bool {
		return atlaspacket.CodeConfigured(opts, "enterError", key)
	}
	defer func() { tradeEnterErrorConfigured = origEnterGate }()

	unlocked = nil
	origUnlock := tradeUnlocker
	tradeUnlocker = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, _ writer.Producer, cid charconst.Id) {
		unlocked = append(unlocked, cid)
	}
	defer func() { tradeUnlocker = origUnlock }()

	origCancel := tradeCancelRequester
	tradeCancelRequester = func(_ logrus.FieldLogger, _ context.Context, f field.Model, cid charconst.Id) error {
		cancelled = append(cancelled, cancelCall{characterId: cid, field: f})
		return nil
	}
	defer func() { tradeCancelRequester = origCancel }()

	d(sc)(l, ctx)
	return recorded
}

// unlocked records the characters whose action lock the handler released.
// Reset by handleAndCaptureWith on every run; the package's tests are not
// parallel.
var unlocked []charconst.Id

// cancelCall captures one invocation of the tradeCancelRequester seam.
type cancelCall struct {
	characterId charconst.Id
	field       field.Model
}

// cancelled records the CANCEL commands the handler under test asked
// atlas-trades for. Reset by handleAndCaptureWith on every run; the package's
// tests are not parallel.
var cancelled []cancelCall

// baseEvent fills the envelope fields every handler's guard reads, plus the
// room identity. Body is supplied by each event constructor.
func baseEvent[E any](o ownerId, v visitorId, actor charconst.Id, eventType string, body E) trade2.StatusEvent[E] {
	return trade2.StatusEvent[E]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     testChannelId,
		RoomId:        uuid.New(),
		Handle:        uint32(o),
		RoomType:      3,
		OwnerId:       charconst.Id(o),
		VisitorId:     charconst.Id(v),
		CharacterId:   actor,
		Type:          eventType,
		Body:          body,
	}
}

// itemStagedEvent stages one item from the side at p. The staging character is
// whichever occupant sits at p, so the event is self-consistent with the
// absolute position the body carries.
func itemStagedEvent(o ownerId, v visitorId, p position) eventDispatch {
	stager := charconst.Id(o)
	if byte(p) == visitorPosition {
		stager = charconst.Id(v)
	}
	e := baseEvent(o, v, stager, trade2.StatusTypeItemStaged, trade2.ItemStagedEventBody{
		Position:  byte(p),
		TradeSlot: testTradeSlot,
		AssetId:   1,
		Snapshot:  sharedsaga.AssetSnapshot{TemplateId: 2000000, Quantity: 50},
	})
	// No resolver stub. The handler renders the event's own snapshot, and that
	// is the point: the stub this replaced faked a successful compartment
	// read-back, so it kept passing while the real read-back — against a
	// compartment the item had already left — missed every time and cancelled
	// the trade.
	return func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) {
			handleItemStagedEvent(sc, nil)(l, ctx, e)
		}
	}
}

func mesoStagedEvent(o ownerId, v visitorId, p position) eventDispatch {
	stager := charconst.Id(o)
	if byte(p) == visitorPosition {
		stager = charconst.Id(v)
	}
	e := baseEvent(o, v, stager, trade2.StatusTypeMesoStaged, trade2.MesoStagedEventBody{
		Position: byte(p),
		Amount:   1234,
	})
	return func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) { handleMesoStagedEvent(sc, nil)(l, ctx, e) }
	}
}

func attestationRequestedEvent(o ownerId, v visitorId) eventDispatch {
	e := baseEvent(o, v, charconst.Id(v), trade2.StatusTypeAttestationRequested, trade2.AttestationRequestedEventBody{})
	return func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) {
			handleAttestationRequestedEvent(sc, nil)(l, ctx, e)
		}
	}
}

// mesoRefusedEvent refuses c's stage in a room that HAS a counterparty (200),
// so "to the refused character only" is actually observable — a room with no
// visitor would make that assertion vacuous.
func mesoRefusedEvent(c characterId, lv lastValid) eventDispatch {
	e := baseEvent(ownerId(c), visitorId(200), charconst.Id(c), trade2.StatusTypeMesoRefused, trade2.MesoRefusedEventBody{
		Position:        ownerPosition,
		LastValidAmount: uint32(lv),
	})
	return func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) { handleMesoRefusedEvent(sc, nil)(l, ctx, e) }
	}
}

func settledEvent(o ownerId, v visitorId) eventDispatch {
	return settledEventOn(o, v, testChannelId)
}

// settledEventOnChannel builds a SETTLED event addressed to a DIFFERENT
// channel than the test server owns, to exercise the guard.
func settledEventOnChannel(ch byte) eventDispatch {
	return settledEventOn(ownerId(100), visitorId(200), ch)
}

func settledEventOn(o ownerId, v visitorId, ch byte) eventDispatch {
	e := baseEvent(o, v, charconst.Id(o), trade2.StatusTypeSettled, trade2.SettledEventBody{LedgerEntryId: uuid.New()})
	e.ChannelId = channelconst.Id(ch)
	return func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) { handleSettledEvent(sc, nil)(l, ctx, e) }
	}
}

// withStubVisitors swaps the visitor-list seam for one that names each entry
// "C<characterId>" over a zero-value avatar, so an enter-result frame can be
// round-tripped without a REST lookup.
func withStubVisitors(d func(sc server.Model) func(logrus.FieldLogger, context.Context)) eventDispatch {
	return func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) {
			orig := tradeRoomVisitorResolver
			tradeRoomVisitorResolver = func(_ logrus.FieldLogger, _ context.Context, slot byte, cid charconst.Id) (interactionpkt.Visitor, error) {
				return interactionpkt.NewBaseVisitor(slot, packetmodel.Avatar{}, fmt.Sprintf("C%d", cid)), nil
			}
			defer func() { tradeRoomVisitorResolver = orig }()
			d(sc)(l, ctx)
		}
	}
}

func roomCreatedEvent(o ownerId) eventDispatch {
	e := baseEvent(o, visitorId(0), charconst.Id(o), trade2.StatusTypeRoomCreated, trade2.RoomCreatedEventBody{Position: ownerPosition})
	return withStubVisitors(func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) { handleRoomCreatedEvent(sc, nil)(l, ctx, e) }
	})
}

func participantEnteredEvent(o ownerId, v visitorId) eventDispatch {
	e := baseEvent(o, v, charconst.Id(v), trade2.StatusTypeParticipantEntered, trade2.ParticipantEnteredEventBody{
		CharacterId: charconst.Id(v),
		Name:        "Partner",
		Position:    visitorPosition,
	})
	return withStubVisitors(func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) {
			handleParticipantEnteredEvent(sc, nil)(l, ctx, e)
		}
	})
}

// inviteSentEvent addresses the invite at target, which the zero case uses to
// exercise announceTo's "0 is not a character id" drop.
func inviteSentEvent(o ownerId, target charconst.Id) eventDispatch {
	e := baseEvent(o, visitorId(0), charconst.Id(o), trade2.StatusTypeInviteSent, trade2.InviteSentEventBody{
		TargetCharacterId: target,
		InviterName:       "Inviter",
	})
	return func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) { handleInviteSentEvent(sc, nil)(l, ctx, e) }
	}
}

func inviteRejectedEvent(o ownerId, code string, targetName string) eventDispatch {
	e := baseEvent(o, visitorId(0), charconst.Id(o), trade2.StatusTypeInviteRejected, trade2.InviteRejectedEventBody{
		Code:       code,
		TargetName: targetName,
	})
	return func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) {
			handleInviteRejectedEvent(sc, nil)(l, ctx, e)
		}
	}
}

// decodeEnterResultRoom decodes the room blob that follows the ENTER_RESULT
// mode byte of a captured frame.
func decodeEnterResultRoom(t *testing.T, c announceCall) interactionpkt.Room {
	t.Helper()
	if got, want := c.bytes[0], modeByte(t, "ENTER_RESULT"); got != want {
		t.Fatalf("frame mode: got %d, want ENTER_RESULT (%d)", got, want)
	}
	l, _ := testlog.NewNullLogger()
	// The avatar codec is version-gated, so the frame must be decoded under the
	// same tenant version it was encoded under (handleAndCapture's GMS 83.1);
	// only the region/version matter, not the tenant's identity.
	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	// NewRequestReader's second argument is a timestamp, not an offset, so the
	// mode byte is sliced off rather than skipped.
	req := request.Request(c.bytes[1:])
	reader := request.NewRequestReader(&req, 0)
	var rm interactionpkt.Room
	rm.Decode(l, ctx)(&reader, testOptions)
	return rm
}

func cancelledEvent(o ownerId, v visitorId, reason string) eventDispatch {
	e := baseEvent(o, v, charconst.Id(o), trade2.StatusTypeCancelled, trade2.CancelledEventBody{Reason: reason})
	return func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) { handleCancelledEvent(sc, nil)(l, ctx, e) }
	}
}

// callsTo returns every announce addressed to one character.
func callsTo(announced []announceCall, cid charconst.Id) []announceCall {
	var out []announceCall
	for _, c := range announced {
		if c.characterId == cid {
			out = append(out, c)
		}
	}
	return out
}

// modeByte resolves a semantic operations KEY to the byte the test table maps
// it to, so an assertion never spells a wire value itself.
func modeByte(t *testing.T, key string) byte {
	t.Helper()
	raw, ok := testOperations[key]
	if !ok {
		t.Fatalf("operations key %q missing from the test table", key)
	}
	return byte(raw.(float64))
}

// sideByteSentTo returns the recipient-relative side byte of the single staged
// frame (TRADE_PUT_ITEM or TRADE_ADD_MESO) sent to one character. Both arms
// carry the side at offset 1, immediately after the mode.
func sideByteSentTo(t *testing.T, announced []announceCall, cid charconst.Id) byte {
	t.Helper()
	cs := callsTo(announced, cid)
	if len(cs) != 1 {
		t.Fatalf("expected exactly one staged frame for character %d, got %d", cid, len(cs))
	}
	b := cs[0].bytes
	if len(b) < 2 {
		t.Fatalf("staged frame for character %d is %d bytes, too short to carry a side", cid, len(b))
	}
	if got := b[0]; got != modeByte(t, "TRADE_PUT_ITEM") && got != modeByte(t, "TRADE_ADD_MESO") {
		t.Fatalf("frame for character %d has mode %d, not a staged-item or staged-meso arm", cid, got)
	}
	return b[1]
}

// receivedMode reports whether the character received at least one frame whose
// mode byte is the one the operations table maps that KEY to.
func receivedMode(t *testing.T, announced []announceCall, cid charconst.Id, key string) bool {
	t.Helper()
	want := modeByte(t, key)
	for _, c := range callsTo(announced, cid) {
		if len(c.bytes) > 0 && c.bytes[0] == want {
			return true
		}
	}
	return false
}

// assertLeaveReason requires exactly one LEAVE frame for the character, and
// that its status byte is the one the leaveReason table maps the KEY to.
func assertLeaveReason(t *testing.T, announced []announceCall, cid charconst.Id, reason string) {
	t.Helper()
	cs := callsTo(announced, cid)
	if len(cs) != 1 {
		t.Fatalf("expected exactly one leave frame for character %d, got %d", cid, len(cs))
	}
	b := cs[0].bytes
	if len(b) != 3 {
		t.Fatalf("leave frame for character %d is %d bytes, want mode+slot+status", cid, len(b))
	}
	if got, want := b[0], modeByte(t, "LEAVE"); got != want {
		t.Errorf("character %d frame mode: got %d, want LEAVE (%d)", cid, got, want)
	}
	raw, ok := testLeaveReason[reason]
	if !ok {
		t.Fatalf("leaveReason key %q missing from the test table", reason)
	}
	if got, want := b[2], byte(raw.(float64)); got != want {
		t.Errorf("character %d leave status: got %d, want %s (%d)", cid, got, reason, want)
	}
}

// TestItemStagedIsSideRelativePerRecipient pins the one thing this consumer
// must get right: the wire `side` byte is RECIPIENT-relative (0 = your own
// side, 1 = the counterparty), while the event carries an absolute room
// position. Sending the same byte to both clients puts the item on the wrong
// side of one of the two dialogs.
func TestItemStagedIsSideRelativePerRecipient(t *testing.T) {
	announced := handleAndCapture(t, itemStagedEvent(ownerId(100), visitorId(200), position(0)))

	if got := sideByteSentTo(t, announced, 100); got != 0 {
		t.Errorf("stager's own side byte: got %d, want 0", got)
	}
	if got := sideByteSentTo(t, announced, 200); got != 1 {
		t.Errorf("counterparty's side byte: got %d, want 1", got)
	}
}

// TestMesoStagedIsSideRelativePerRecipient pins the same for the staged-meso
// arm, staged from the VISITOR's side so the two recipients' bytes are the
// mirror of the item case.
func TestMesoStagedIsSideRelativePerRecipient(t *testing.T) {
	announced := handleAndCapture(t, mesoStagedEvent(ownerId(100), visitorId(200), position(1)))
	if got := sideByteSentTo(t, announced, 200); got != 0 {
		t.Errorf("stager's own side byte: got %d, want 0", got)
	}
	if got := sideByteSentTo(t, announced, 100); got != 1 {
		t.Errorf("counterparty's side byte: got %d, want 1", got)
	}
}

// TestAttestationRequestedGoesToBothSides pins design §6.2: the attestation
// prompt is broadcast to BOTH clients at once, only after both confirmed.
func TestAttestationRequestedGoesToBothSides(t *testing.T) {
	announced := handleAndCapture(t, attestationRequestedEvent(ownerId(100), visitorId(200)))
	for _, id := range []charconst.Id{100, 200} {
		if !receivedMode(t, announced, id, "TRADE_CONFIRM") {
			t.Errorf("character %d did not receive the attestation prompt", id)
		}
	}
}

// TestMesoRefusedSendsBothTheReEchoAndTheLimitArm pins design §4.2: the
// authoritative TRADE_ADD_MESO re-echo is what actually corrects the client
// (that arm is an ASSIGNMENT), and TRADE_MESO_LIMIT only supplies the reason.
func TestMesoRefusedSendsBothTheReEchoAndTheLimitArm(t *testing.T) {
	announced := handleAndCapture(t, mesoRefusedEvent(characterId(100), lastValid(1_000_000)))
	if !receivedMode(t, announced, 100, "TRADE_ADD_MESO") {
		t.Error("no authoritative re-echo sent")
	}
	if !receivedMode(t, announced, 100, "TRADE_MESO_LIMIT") {
		t.Error("no meso-limit reason sent")
	}
	// The counterparty never staged anything and must not be told otherwise:
	// the re-echo carries the refused side's own amount, so leaking it to the
	// other client would rewrite that client's view of its own grid.
	if got := len(callsTo(announced, 200)); got != 0 {
		t.Errorf("counterparty received %d frames, want 0", got)
	}
}

// TestMesoRefusedSkipsTheLimitArmWhereItIsVersionAbsent pins the jms_185 case:
// CTradingRoomDlg there has no meso-limit arm, so template_jms_185_1.json binds
// no TRADE_MESO_LIMIT operation. Sending it anyway would resolve to the 99
// sentinel and hand the client a mode it cannot dispatch. The authoritative
// TRADE_ADD_MESO re-echo must still go out — it is what actually corrects the
// client, per design §4.2.
func TestMesoRefusedSkipsTheLimitArmWhereItIsVersionAbsent(t *testing.T) {
	ops := make(map[string]interface{}, len(testOperations))
	for k, v := range testOperations {
		if k == "TRADE_MESO_LIMIT" {
			continue
		}
		ops[k] = v
	}
	opts := map[string]interface{}{"operations": ops, "leaveReason": testLeaveReason}

	announced := handleAndCaptureWith(t, mesoRefusedEvent(characterId(100), lastValid(1_000_000)), opts)
	if !receivedMode(t, announced, 100, "TRADE_ADD_MESO") {
		t.Error("the authoritative re-echo must still be sent where the limit arm is absent")
	}
	if got := len(callsTo(announced, 100)); got != 1 {
		t.Errorf("character received %d interaction frames, want exactly 1 (the re-echo alone)", got)
	}
	// The unlock is a StatChanged, not a CharacterInteraction body, so it is
	// not one of the frames counted above — but it must still have been sent.
	if len(unlocked) != 1 || unlocked[0] != 100 {
		t.Errorf("expected the refused character to be unlocked, got %v", unlocked)
	}
}

// TestMesoRefusedReleasesTheActionLock is the load-bearing test of design
// §5A.6.
//
// CTradingRoomDlg::PutMoney arms m_bExclRequestSent on send and
// CanSendExclRequest refuses every later PUT_ITEM and PUT_MONEY until a server
// packet clears it. A refusal produces no meso mutation, so nothing clears it
// implicitly: without an explicit unlock the player's mesos button and every
// further item stage silently stop working for the rest of the session. That is
// the exact bug reported against the reserve-at-staging build.
func TestMesoRefusedReleasesTheActionLock(t *testing.T) {
	handleAndCapture(t, mesoRefusedEvent(characterId(100), lastValid(1_000_000)))
	if len(unlocked) != 1 {
		t.Fatalf("expected exactly one unlock, got %d", len(unlocked))
	}
	if unlocked[0] != 100 {
		t.Errorf("unlocked character %d, want the refused character 100", unlocked[0])
	}
}

// TestMesoStagedDoesNotSendAnExplicitUnlock pins the converse. A SUCCESSFUL
// meso stage is a real debit, so atlas-character publishes a STAT_CHANGED whose
// own exclRequestSent bool clears the lock. Sending a second, empty unlock here
// would be redundant at best; the rule is that only outcomes with no mutation
// unlock explicitly.
func TestMesoStagedDoesNotSendAnExplicitUnlock(t *testing.T) {
	handleAndCapture(t, mesoStagedEvent(ownerId(100), visitorId(200), position(0)))
	if len(unlocked) != 0 {
		t.Errorf("a successful stage must not send an explicit unlock, got %v", unlocked)
	}
}

// TestSettledSendsLeaveSuccessToBothSides pins design §1.4: completion is
// LEAVE + slot + the success status, not a distinct mode.
func TestSettledSendsLeaveSuccessToBothSides(t *testing.T) {
	announced := handleAndCapture(t, settledEvent(ownerId(100), visitorId(200)))
	for _, id := range []charconst.Id{100, 200} {
		assertLeaveReason(t, announced, id, interactioncb.CharacterInteractionLeaveReasonTradeSuccess)
	}
}

// TestCancelledMapsTheReasonKeyStraightThrough pins DOM-25: the event carries a
// semantic KEY and the channel resolves it via the tenant leaveReason table —
// the channel never invents a numeric status.
func TestCancelledMapsTheReasonKeyStraightThrough(t *testing.T) {
	for _, reason := range []string{"TRADE_CANCELLED", "TRADE_FAILED", "TRADE_CANNOT_CARRY", "TRADE_DIFFERENT_MAP", "TRADE_CRC_FAILED"} {
		t.Run(reason, func(t *testing.T) {
			announced := handleAndCapture(t, cancelledEvent(ownerId(100), visitorId(200), reason))
			assertLeaveReason(t, announced, 100, reason)
		})
	}
}

// TestEventsForAnotherChannelAreIgnored pins the guard every handler runs.
func TestEventsForAnotherChannelAreIgnored(t *testing.T) {
	announced := handleAndCapture(t, settledEventOnChannel(9))
	if len(announced) != 0 {
		t.Errorf("handled an event for another channel: %d writes", len(announced))
	}
}

// TestSettledAddressesEachRecipientWithItsOwnSeat pins that the LEAVE slot byte
// is the recipient's own absolute room position, not one seat sent to both.
func TestSettledAddressesEachRecipientWithItsOwnSeat(t *testing.T) {
	announced := handleAndCapture(t, settledEvent(ownerId(100), visitorId(200)))
	for _, tc := range []struct {
		cid  charconst.Id
		seat byte
	}{{100, ownerPosition}, {200, visitorPosition}} {
		cs := callsTo(announced, tc.cid)
		if len(cs) != 1 {
			t.Fatalf("expected one leave frame for character %d, got %d", tc.cid, len(cs))
		}
		if got := cs[0].bytes[1]; got != tc.seat {
			t.Errorf("character %d leave slot: got %d, want %d", tc.cid, got, tc.seat)
		}
	}
}

// TestRoomCreatedCarriesTheOwnerEntryAtSlotZero pins the enter-result shape the
// reviewer had to establish by decompilation:
// CMiniRoomBaseDlg::OnEnterResultBase (@0x65ec3d) populates the dialog's avatar
// array EXCLUSIVELY from this visitor list — nothing seeds it from local
// CharacterData — and ::OnLeaveBase (@0x65edb5) opens with a
// CDisconnectException throw on a LEAVE naming a slot the array never filled.
// An empty list here is therefore not a cosmetic gap: the very next
// SETTLED/CANCELLED LEAVE for slot 0 disconnects the owner's client.
func TestRoomCreatedCarriesTheOwnerEntryAtSlotZero(t *testing.T) {
	announced := handleAndCapture(t, roomCreatedEvent(ownerId(100)))
	cs := callsTo(announced, 100)
	if len(cs) != 1 {
		t.Fatalf("expected one enter-result frame for the owner, got %d", len(cs))
	}
	rm := decodeEnterResultRoom(t, cs[0])
	if got := len(rm.Visitors()); got != 1 {
		t.Fatalf("visitor count: got %d, want 1 (the owner's own entry)", got)
	}
	if got := rm.Visitors()[0].Slot(); got != ownerPosition {
		t.Errorf("owner entry slot: got %d, want %d", got, ownerPosition)
	}
	if got := rm.Position(); got != ownerPosition {
		t.Errorf("header position: got %d, want the owner's own seat %d", got, ownerPosition)
	}
}

// TestParticipantEnteredGivesTheEntrantBothSeats pins the other half of the same
// invariant: the entrant's dialog must know about BOTH occupants, or a later
// LEAVE naming the owner's slot 0 hits the same disconnect branch. The header
// position is the entrant's OWN seat, which is what the dialog branches on.
func TestParticipantEnteredGivesTheEntrantBothSeats(t *testing.T) {
	announced := handleAndCapture(t, participantEnteredEvent(ownerId(100), visitorId(200)))

	cs := callsTo(announced, 200)
	if len(cs) != 1 {
		t.Fatalf("expected one enter-result frame for the entrant, got %d", len(cs))
	}
	rm := decodeEnterResultRoom(t, cs[0])
	if got := len(rm.Visitors()); got != 2 {
		t.Fatalf("visitor count: got %d, want 2", got)
	}
	for i, want := range []byte{ownerPosition, visitorPosition} {
		if got := rm.Visitors()[i].Slot(); got != want {
			t.Errorf("visitor[%d] slot: got %d, want %d", i, got, want)
		}
	}
	if got := rm.Position(); got != visitorPosition {
		t.Errorf("header position: got %d, want the entrant's own seat %d", got, visitorPosition)
	}

	// The owner, whose dialog is already open, gets the incremental ENTER.
	if !receivedMode(t, announced, 100, "ENTER") {
		t.Error("the owner did not receive the incremental ENTER for the new visitor")
	}
}

// TestInviteRejectedPassesTheTargetNameThrough pins that the refused target's
// name reaches the wire. GMS v83 CMiniRoomBaseDlg::OnInviteResultStatic
// (@0x65E848) DecodeStr's a name for codes 2/3/4 and Formats it into the
// message, so dropping it renders "%s is doing something else right now" with a
// blank subject. The INVITE_RESULT frame is mode + code + AsciiString, and the
// string is length-prefixed with a uint16.
func TestInviteRejectedPassesTheTargetNameThrough(t *testing.T) {
	announced := handleAndCapture(t, inviteRejectedEvent(ownerId(100), "BUSY", "Guest"))
	cs := callsTo(announced, 100)
	if len(cs) != 1 {
		t.Fatalf("expected one invite-result frame for the inviter, got %d", len(cs))
	}
	b := cs[0].bytes
	if got, want := b[0], modeByte(t, "INVITE_RESULT"); got != want {
		t.Errorf("frame mode: got %d, want INVITE_RESULT (%d)", got, want)
	}
	if got, want := string(b[4:]), "Guest"; got != want {
		t.Errorf("target name on the wire: got %q, want %q", got, want)
	}
}

// TestAnnounceToDropsCharacterZero pins announceTo's guard: zero is not a
// character id, and a frame addressed to it can only be a session lookup that
// misses.
func TestAnnounceToDropsCharacterZero(t *testing.T) {
	if got := len(handleAndCapture(t, inviteSentEvent(ownerId(100), 0))); got != 0 {
		t.Errorf("announced %d frames to character 0, want 0", got)
	}
	if got := len(handleAndCapture(t, inviteSentEvent(ownerId(100), 200))); got != 1 {
		t.Fatalf("a real target must still be announced to: got %d frames, want 1", got)
	}
}

// TestStatusTypeIsCheckedBeforeTheBodyIsActedOn pins the shared-topic rule: the
// status topic fans out to every handler, so a handler whose type does not
// match must write nothing even though the envelope decoded cleanly.
func TestStatusTypeIsCheckedBeforeTheBodyIsActedOn(t *testing.T) {
	announced := handleAndCapture(t, func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		e := baseEvent(ownerId(100), visitorId(200), 100, trade2.StatusTypeCancelled, trade2.SettledEventBody{})
		return func(l logrus.FieldLogger, ctx context.Context) { handleSettledEvent(sc, nil)(l, ctx, e) }
	})
	if len(announced) != 0 {
		t.Errorf("the settled handler acted on a CANCELLED event: %d writes", len(announced))
	}
}

// --- FIX 1: the enterError crash sentinel -------------------------------------

// errorEvent is an ERROR status addressed at c, carrying an enterError KEY.
func errorEvent(c characterId, code string) eventDispatch {
	e := baseEvent(ownerId(0), visitorId(0), charconst.Id(c), trade2.StatusTypeError, trade2.ErrorEventBody{Code: code})
	return func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) { handleErrorEvent(sc, nil)(l, ctx, e) }
	}
}

// TestErrorEventSendsARefusalWhoseCodeIsBound is the positive half: on a tenant
// whose enterError table binds the key, the refusal goes out with the table's
// byte.
func TestErrorEventSendsARefusalWhoseCodeIsBound(t *testing.T) {
	announced := handleAndCapture(t, errorEvent(characterId(100), "OTHER_REQUESTS"))
	cs := callsTo(announced, 100)
	if len(cs) != 1 {
		t.Fatalf("expected one enter-result frame, got %d", len(cs))
	}
	// The frame is mode + a zero pad + the refusal code
	// (InteractionEnterResultError.Encode), both bytes tenant-resolved.
	if got, want := cs[0].bytes[0], byte(5); got != want {
		t.Errorf("mode byte: got %d, want ENTER_RESULT %d", got, want)
	}
	if got, want := cs[0].bytes[2], byte(3); got != want {
		t.Errorf("refusal byte: got %d, want the table's OTHER_REQUESTS %d", got, want)
	}
}

// TestErrorEventSkipsARefusalWhoseCodeIsUnbound pins the crash-sentinel guard
// against the SHIPPED tables. template_gms_92_1.json and template_gms_12_1.json
// bind no `enterError` property at all, and template_jms_185_1.json binds only
// TRADE_NOT_ALLOWED / TRADE_NOT_ALLOWED_2 — while all three DO bind
// operations.ENTER_RESULT, so the frame dispatches and only the payload byte is
// garbage. atlas-trades emits six keys (ROOM_CLOSED, OTHER_REQUESTS,
// NOT_WHEN_DEAD, UNABLE, TRADE_NOT_ALLOWED, NOT_SAME_MAP), so without the gate
// every trade refusal on a v92 tenant, and five of six on jms, would write
// ResolveCode's 99 sentinel — documented in libs/atlas-packet/resolve.go as
// "will likely cause a client crash".
func TestErrorEventSkipsARefusalWhoseCodeIsUnbound(t *testing.T) {
	// A tenant with the mode bound but NO enterError table — the gms_v92 shape.
	opts := map[string]interface{}{"operations": testOperations, "leaveReason": testLeaveReason}
	for _, code := range []string{"ROOM_CLOSED", "OTHER_REQUESTS", "NOT_WHEN_DEAD", "UNABLE", "TRADE_NOT_ALLOWED", "NOT_SAME_MAP"} {
		t.Run(code, func(t *testing.T) {
			announced := handleAndCaptureWith(t, errorEvent(characterId(100), code), opts)
			if got := len(announced); got != 0 {
				t.Fatalf("frames sent: got %d, want 0 — an unbound key must not reach the 99 sentinel (bytes %v)", got, announced[0].bytes)
			}
		})
	}
}

// TestErrorEventStillSendsTheKeysJmsDoesBind pins that the gate is per-KEY and
// not per-tenant: jms_185 binds TRADE_NOT_ALLOWED, and that refusal must still
// reach the client there.
func TestErrorEventStillSendsTheKeysJmsDoesBind(t *testing.T) {
	jmsEnterError := map[string]interface{}{
		"TRADE_NOT_ALLOWED":   float64(7),
		"TRADE_NOT_ALLOWED_2": float64(20),
	}
	opts := map[string]interface{}{"operations": testOperations, "leaveReason": testLeaveReason, "enterError": jmsEnterError}

	if got := len(handleAndCaptureWith(t, errorEvent(characterId(100), "TRADE_NOT_ALLOWED"), opts)); got != 1 {
		t.Errorf("TRADE_NOT_ALLOWED frames: got %d, want 1", got)
	}
	if got := len(handleAndCaptureWith(t, errorEvent(characterId(100), "NOT_SAME_MAP"), opts)); got != 0 {
		t.Errorf("NOT_SAME_MAP frames: got %d, want 0 — jms binds no code for it", got)
	}
}

// --- FIX 2 / FIX 4: a display failure must not leave the room live ------------

// errDisplay is the transient REST failure the visitor resolver seam raises —
// the only display lookup left in this consumer now that the staged item's frame
// is built from the event's own snapshot.
var errDisplay = errors.New("compartment service unavailable")

// snapshotStagedEvent stages one item described entirely by the snapshot the
// event carries — the only source the handler has, since the asset has already
// left its owner's compartment for escrow.
func snapshotStagedEvent(o ownerId, v visitorId, s sharedsaga.AssetSnapshot) eventDispatch {
	e := baseEvent(o, v, charconst.Id(o), trade2.StatusTypeItemStaged, trade2.ItemStagedEventBody{
		Position:  ownerPosition,
		TradeSlot: testTradeSlot,
		AssetId:   1,
		Snapshot:  s,
	})
	return func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) {
			handleItemStagedEvent(sc, nil)(l, ctx, e)
		}
	}
}

// withFailingVisitors is withStubVisitors' negative twin.
func withFailingVisitors(d func(sc server.Model) func(logrus.FieldLogger, context.Context)) eventDispatch {
	return func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) {
			orig := tradeRoomVisitorResolver
			tradeRoomVisitorResolver = func(logrus.FieldLogger, context.Context, byte, charconst.Id) (interactionpkt.Visitor, error) {
				return interactionpkt.Visitor{}, errDisplay
			}
			defer func() { tradeRoomVisitorResolver = orig }()
			d(sc)(l, ctx)
		}
	}
}

func failingRoomCreatedEvent(o ownerId) eventDispatch {
	e := baseEvent(o, visitorId(0), charconst.Id(o), trade2.StatusTypeRoomCreated, trade2.RoomCreatedEventBody{Position: ownerPosition})
	return withFailingVisitors(func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) { handleRoomCreatedEvent(sc, nil)(l, ctx, e) }
	})
}

func failingParticipantEnteredEvent(o ownerId, v visitorId) eventDispatch {
	e := baseEvent(o, v, charconst.Id(v), trade2.StatusTypeParticipantEntered, trade2.ParticipantEnteredEventBody{
		CharacterId: charconst.Id(v),
		Name:        "Partner",
		Position:    visitorPosition,
	})
	return withFailingVisitors(func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) {
			handleParticipantEnteredEvent(sc, nil)(l, ctx, e)
		}
	})
}

// TestStagedItemNeverCancelsTheTrade is the inverse of the display-failure test
// this replaced, and it is the regression guard for the escrow defect.
//
// Under escrow-at-staging the asset has already left its owner's compartment
// when ITEM_STAGED fires, so the compartment read-back this handler used to
// perform could ONLY miss. Every item-carrying trade therefore self-cancelled on
// its first staged item. The frame is now built from the snapshot the event
// carries: there is no lookup left, hence no failure branch, hence no cancel.
//
// It runs over the plain stackable, the cash item and the pet because the three
// take different arms of the item encoder, and a snapshot that fed one arm
// garbage would previously have shown up only as a mis-rendered dialog.
func TestStagedItemNeverCancelsTheTrade(t *testing.T) {
	for name, s := range map[string]sharedsaga.AssetSnapshot{
		"stackable": {TemplateId: 2000000, Quantity: 50},
		"cash item": {TemplateId: 5040000, Quantity: 1, CashId: stagedCashId},
		"pet":       {TemplateId: 5000000, Quantity: 1, CashId: stagedCashId, PetId: 909, PetName: stagedPetName, PetLevel: 3, Closeness: 450, Fullness: 88},
	} {
		t.Run(name, func(t *testing.T) {
			announced := handleAndCapture(t, snapshotStagedEvent(ownerId(100), visitorId(200), s))
			if got := len(announced); got != 2 {
				t.Errorf("frames sent: got %d, want one per occupant (2)", got)
			}
			if len(cancelled) != 0 {
				t.Fatalf("CANCEL commands: got %d, want 0 — rendering a staged item cannot fail any more", len(cancelled))
			}
		})
	}
}

// TestStagedCashItemAndPetKeepTheirIdentity pins the second half of the escrow
// defect: the snapshot must reach the wire with its cash serial and pet state
// intact.
//
// Cash items and pets are stageable — atlas-trades' checkRestrictions blocks
// equipped items, the untradeable flags and the WZ tradeBlock, and nothing about
// the cash inventory — so a carrier that dropped these fields degraded a real
// player's item. The assertion is on the ENCODED BYTES rather than on a struct
// field, because the point is that the value survives all the way into the frame
// both clients read.
func TestStagedCashItemAndPetKeepTheirIdentity(t *testing.T) {
	cash := make([]byte, 8)
	binary.LittleEndian.PutUint64(cash, uint64(stagedCashId))

	t.Run("cash item keeps its serial", func(t *testing.T) {
		announced := handleAndCapture(t, snapshotStagedEvent(ownerId(100), visitorId(200),
			sharedsaga.AssetSnapshot{TemplateId: 5040000, Quantity: 1, CashId: stagedCashId}))
		if len(announced) == 0 {
			t.Fatal("no frames announced")
		}
		if !bytes.Contains(announced[0].bytes, cash) {
			t.Errorf("the staged cash item's serial %d is not in the frame: % x", stagedCashId, announced[0].bytes)
		}
	})

	t.Run("pet keeps its name and serial", func(t *testing.T) {
		announced := handleAndCapture(t, snapshotStagedEvent(ownerId(100), visitorId(200),
			sharedsaga.AssetSnapshot{
				TemplateId: 5000000, Quantity: 1, CashId: stagedCashId,
				PetId: 909, PetName: stagedPetName, PetLevel: 3, Closeness: 450, Fullness: 88,
			}))
		if len(announced) == 0 {
			t.Fatal("no frames announced")
		}
		if !bytes.Contains(announced[0].bytes, []byte(stagedPetName)) {
			t.Errorf("the staged pet's name %q is not in the frame: % x", stagedPetName, announced[0].bytes)
		}
		// A pet's wire serial is its PET id, not its cash id: the pet block writes
		// PetSerialNumber, which falls back to petId when no explicit serial was
		// set (packetmodel Asset.PetSerialNumber). The snapshot carries no pet
		// serial, so petId is what identifies the pet to the client — losing it
		// would render an unidentifiable pet.
		petSerial := make([]byte, 8)
		binary.LittleEndian.PutUint64(petSerial, 909)
		if !bytes.Contains(announced[0].bytes, petSerial) {
			t.Errorf("the staged pet's serial (petId 909) is not in the frame: % x", announced[0].bytes)
		}
	})
}

const (
	stagedCashId  = int64(4815162342)
	stagedPetName = "Fluffy"
)

// TestRoomCreatedDisplayFailureCancelsTheRoom pins the soft-lock. atlas-trades
// has already seated the owner, so without a teardown they can neither create
// another room (ErrOwnerHasRoom) nor enter a mini-game, and they have no dialog
// to close, so no CANCEL ever comes from the client. It self-heals only on map
// change, channel change or logout.
func TestRoomCreatedDisplayFailureCancelsTheRoom(t *testing.T) {
	announced := handleAndCapture(t, failingRoomCreatedEvent(ownerId(100)))
	if got := len(announced); got != 0 {
		t.Errorf("frames sent: got %d, want 0", got)
	}
	if len(cancelled) != 1 {
		t.Fatalf("CANCEL commands: got %d, want 1", len(cancelled))
	}
	if cancelled[0].characterId != 100 {
		t.Errorf("CANCEL addressed character %d, want the owner 100", cancelled[0].characterId)
	}
}

// TestParticipantEnteredDisplayFailureCancelsTheRoom is the same shape for the
// entrant: seated server-side, no dialog either side can act on.
func TestParticipantEnteredDisplayFailureCancelsTheRoom(t *testing.T) {
	announced := handleAndCapture(t, failingParticipantEnteredEvent(ownerId(100), visitorId(200)))
	if got := len(announced); got != 0 {
		t.Errorf("frames sent: got %d, want 0", got)
	}
	if len(cancelled) != 1 {
		t.Fatalf("CANCEL commands: got %d, want 1", len(cancelled))
	}
	if cancelled[0].characterId != 200 {
		t.Errorf("CANCEL addressed character %d, want the entrant 200", cancelled[0].characterId)
	}
}

// TestASuccessfulDisplayCancelsNothing is the mutation guard for all three:
// the happy path must not tear rooms down.
func TestASuccessfulDisplayCancelsNothing(t *testing.T) {
	for name, d := range map[string]eventDispatch{
		"item staged":         itemStagedEvent(ownerId(100), visitorId(200), position(ownerPosition)),
		"room created":        roomCreatedEvent(ownerId(100)),
		"participant entered": participantEnteredEvent(ownerId(100), visitorId(200)),
	} {
		t.Run(name, func(t *testing.T) {
			handleAndCapture(t, d)
			if len(cancelled) != 0 {
				t.Errorf("CANCEL commands: got %d, want 0", len(cancelled))
			}
		})
	}
}
