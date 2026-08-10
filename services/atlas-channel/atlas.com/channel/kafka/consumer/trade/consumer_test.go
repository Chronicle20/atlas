package trade

import (
	trade2 "atlas-channel/kafka/message/trade"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	interactioncb "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
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

// testOptions is the full writer option set a captured body is encoded with.
var testOptions = map[string]interface{}{
	"operations":  testOperations,
	"leaveReason": testLeaveReason,
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
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	l, _ := testlog.NewNullLogger()
	var recorded []announceCall
	orig := tradeAnnouncer
	tradeAnnouncer = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, _ writer.Producer, cid charconst.Id, body packet.Encode) {
		recorded = append(recorded, announceCall{characterId: cid, bytes: body(l, ctx)(testOptions)})
	}
	defer func() { tradeAnnouncer = orig }()

	d(sc)(l, ctx)
	return recorded
}

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
		Position:   byte(p),
		TradeSlot:  testTradeSlot,
		AssetId:    1,
		TemplateId: 2000000,
		Quantity:   50,
	})
	return func(sc server.Model) func(logrus.FieldLogger, context.Context) {
		return func(l logrus.FieldLogger, ctx context.Context) {
			orig := tradeStagedAssetResolver
			tradeStagedAssetResolver = func(_ logrus.FieldLogger, _ context.Context, _ charconst.Id, b trade2.ItemStagedEventBody) (packetmodel.Asset, error) {
				return packetmodel.NewAsset(true, 0, uint32(b.TemplateId), time.Time{}).SetStackableInfo(uint32(b.Quantity), 0, 0), nil
			}
			defer func() { tradeStagedAssetResolver = orig }()
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

func mesoRefusedEvent(c characterId, lv lastValid) eventDispatch {
	e := baseEvent(ownerId(c), visitorId(0), charconst.Id(c), trade2.StatusTypeMesoRefused, trade2.MesoRefusedEventBody{
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
