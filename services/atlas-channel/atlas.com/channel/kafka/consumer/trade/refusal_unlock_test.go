package trade

import (
	trade2 "atlas-channel/kafka/message/trade"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"bytes"
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	interactioncb "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/clientbound"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	socketwriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// The refusal-unlock suite runs the handlers against LIVE sessions and a REAL
// writer producer, rather than through the announcer/unlocker seams the rest of
// consumer_test.go uses.
//
// The seams answer "was the unlock requested"; they cannot answer "what did the
// client actually receive", and for design §5A.6 that is the whole question.
// The unlock is an EMPTY STAT_CHANGED whose leading exclRequestSent bool is the
// only thing that clears CWvsContext::m_bExclRequestSent — an unlock that
// carried a stat payload, or that went out as some other packet, would look
// identical at the seam and be useless at the client.

// unlockOpcode and interactionOpcode are the two writer opcodes this suite
// encodes with. Their values are arbitrary within the test: the assertions are
// about which writer produced a frame and what followed its opcode, not about
// the per-version numbers, which come from the tenant socket config in
// production.
const (
	unlockOpcode      = byte(0x1F)
	interactionOpcode = byte(0x35)
)

// sentFrame is one plaintext frame as the writer produced it, before the
// session encrypted it onto the wire.
type sentFrame struct {
	writerName string
	bytes      []byte
}

// captureConn is the session's socket. It records how many frames were written
// to it, which is what makes "addressed to one character alone" observable: a
// frame is delivered per session, so a counterparty that received nothing wrote
// nothing.
type captureConn struct {
	mutex  sync.Mutex
	writes int
}

func (c *captureConn) Write(b []byte) (int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.writes++
	return len(b), nil
}

func (c *captureConn) count() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.writes
}

func (c *captureConn) Read([]byte) (int, error)         { return 0, net.ErrClosed }
func (c *captureConn) Close() error                     { return nil }
func (c *captureConn) LocalAddr() net.Addr              { return &net.TCPAddr{} }
func (c *captureConn) RemoteAddr() net.Addr             { return &net.TCPAddr{} }
func (c *captureConn) SetDeadline(time.Time) error      { return nil }
func (c *captureConn) SetReadDeadline(time.Time) error  { return nil }
func (c *captureConn) SetWriteDeadline(time.Time) error { return nil }

// liveRoom is a trade room whose two occupants both have a registered session.
type liveRoom struct {
	l      logrus.FieldLogger
	ctx    context.Context
	sc     server.Model
	wp     writer.Producer
	conns  map[charconst.Id]*captureConn
	frames *[]sentFrame
}

// newLiveRoom registers a session per character and builds the writer producer
// the handlers will announce through. Only the two writers the trade refusal
// paths can legitimately use are bound, so a handler reaching for a third would
// fail its announce rather than silently produce an unasserted frame.
func newLiveRoom(t *testing.T, ids ...charconst.Id) *liveRoom {
	t.Helper()

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	l, _ := testlog.NewNullLogger()
	t.Cleanup(func() { session.ClearRegistryForTenant(tm.Id()) })

	var frames []sentFrame
	record := func(name string, opcode byte, options map[string]interface{}) socketwriter.BodyFunc {
		base := socketwriter.MessageGetter(func(w *response.Writer) { w.WriteByte(opcode) }, options)
		return func(fl logrus.FieldLogger, fctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := base(fl, fctx)(encoder)
				frames = append(frames, sentFrame{writerName: name, bytes: b})
				return b
			}
		}
	}
	wp := socketwriter.ProducerGetter(map[string]socketwriter.BodyFunc{
		statpkt.StatChangedWriter:                record(statpkt.StatChangedWriter, unlockOpcode, map[string]interface{}{}),
		interactioncb.CharacterInteractionWriter: record(interactioncb.CharacterInteractionWriter, interactionOpcode, testOptions),
	})

	conns := make(map[charconst.Id]*captureConn, len(ids))
	sp := session.NewProcessor(l, ctx)
	ch := channelconst.NewModel(0, testChannelId)
	for _, id := range ids {
		conn := &captureConn{}
		sessionId := uuid.New()
		sp.Create(ch, 0)(sessionId, conn)
		sp.SetCharacterId(sessionId, uint32(id))
		conns[id] = conn
	}
	// The hello packet each Create wrote is setup, not a handler's output.
	for _, c := range conns {
		c.mutex.Lock()
		c.writes = 0
		c.mutex.Unlock()
	}

	return &liveRoom{l: l, ctx: ctx, sc: sc, wp: wp, conns: conns, frames: &frames}
}

// expectedUnlockFrame is the byte sequence the unlock MUST be: the writer's
// opcode, the exclRequestSent bool set, an EMPTY stat mask, and the single
// trailing flag byte a pre-v95 STAT_CHANGED carries. Written out literally
// rather than derived from statpkt, because a derivation would move with the
// production expression it is supposed to pin.
var expectedUnlockFrame = []byte{unlockOpcode, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00}

// itemRefusedEvent refuses c's stage in a room that HAS a counterparty, so
// "to the refused character alone" is actually observable.
func itemRefusedEvent(c charconst.Id, counterparty charconst.Id) trade2.StatusEvent[trade2.ItemRefusedEventBody] {
	return baseEvent(ownerId(c), visitorId(counterparty), c, trade2.StatusTypeItemRefused, trade2.ItemRefusedEventBody{
		Position:  ownerPosition,
		TradeSlot: testTradeSlot,
	})
}

// TestItemRefusedWritesTheUnlockAndNothingElse is the load-bearing assertion of
// design §5A.6 for the item path.
//
// A refused stage is player-visibly silent: the empty slot is the feedback, so
// there is no trade packet to send. But the client armed m_bExclRequestSent on
// PUT_ITEM and a refusal produces no inventory delta to clear it, so the one
// thing that MUST go out is the bare unlock. Both halves are asserted on the
// encoded bytes: exactly one frame, from the StatChanged writer, carrying an
// empty update list with the bool set.
func TestItemRefusedWritesTheUnlockAndNothingElse(t *testing.T) {
	r := newLiveRoom(t, 100, 200)

	handleItemRefusedEvent(r.sc, r.wp)(r.l, r.ctx, itemRefusedEvent(100, 200))

	frames := *r.frames
	if len(frames) != 1 {
		t.Fatalf("frames written: got %d, want exactly 1 (the unlock alone) — %+v", len(frames), frames)
	}
	if frames[0].writerName != statpkt.StatChangedWriter {
		t.Errorf("writer: got %s, want %s — a refused stage sends no trade packet", frames[0].writerName, statpkt.StatChangedWriter)
	}
	if !bytes.Equal(frames[0].bytes, expectedUnlockFrame) {
		t.Errorf("unlock frame: got % x, want % x (exclRequestSent set, empty stat mask)", frames[0].bytes, expectedUnlockFrame)
	}
}

// TestItemRefusedAddressesTheRefusedCharacterAlone pins the addressing. The
// counterparty never saw the refused item — nothing was announced when it was
// staged, because a pending stage is announced to nobody — so a frame reaching
// that client would be an unlock for a request it never sent.
func TestItemRefusedAddressesTheRefusedCharacterAlone(t *testing.T) {
	r := newLiveRoom(t, 100, 200)

	handleItemRefusedEvent(r.sc, r.wp)(r.l, r.ctx, itemRefusedEvent(100, 200))

	if got := r.conns[100].count(); got != 1 {
		t.Errorf("refused character received %d frames, want 1", got)
	}
	if got := r.conns[200].count(); got != 0 {
		t.Errorf("counterparty received %d frames, want 0", got)
	}
}

// TestMesoRefusedWritesTheSameUnlockBytes is the twin. handleMesoRefusedEvent's
// unlock is already covered at the seam level
// (TestMesoRefusedReleasesTheActionLock), but only as "an unlock was
// requested" — this pins that what reaches the client is the same empty
// STAT_CHANGED, and that it accompanies rather than replaces the authoritative
// re-echo.
//
// It is also this suite's non-vacuity control: the item path's "exactly one
// frame" assertion above is only meaningful because a handler that DOES
// announce produces more than one here.
func TestMesoRefusedWritesTheSameUnlockBytes(t *testing.T) {
	r := newLiveRoom(t, 100, 200)

	e := baseEvent(ownerId(100), visitorId(200), 100, trade2.StatusTypeMesoRefused, trade2.MesoRefusedEventBody{
		Position:        ownerPosition,
		LastValidAmount: 1_000_000,
	})
	// The real arm-presence gate reads the tenant writer registry, which this
	// suite does not stand up; the table the frames are encoded against is the
	// authority here, exactly as in handleAndCaptureWith.
	origGate := tradeMesoLimitConfigured
	tradeMesoLimitConfigured = func(logrus.FieldLogger, context.Context) bool { return true }
	defer func() { tradeMesoLimitConfigured = origGate }()

	handleMesoRefusedEvent(r.sc, r.wp)(r.l, r.ctx, e)

	frames := *r.frames
	var unlocks []sentFrame
	var interactions int
	for _, f := range frames {
		if f.writerName == statpkt.StatChangedWriter {
			unlocks = append(unlocks, f)
			continue
		}
		interactions++
	}
	if len(unlocks) != 1 {
		t.Fatalf("unlock frames: got %d, want 1", len(unlocks))
	}
	if !bytes.Equal(unlocks[0].bytes, expectedUnlockFrame) {
		t.Errorf("unlock frame: got % x, want % x", unlocks[0].bytes, expectedUnlockFrame)
	}
	// The re-echo and the limit arm. Both are CharacterInteraction bodies and
	// neither carries the exclRequestSent bool, which is why the unlock above is
	// needed at all.
	if interactions != 2 {
		t.Errorf("interaction frames: got %d, want 2 (the re-echo and the limit arm)", interactions)
	}
	if got := r.conns[200].count(); got != 0 {
		t.Errorf("counterparty received %d frames, want 0", got)
	}
}
