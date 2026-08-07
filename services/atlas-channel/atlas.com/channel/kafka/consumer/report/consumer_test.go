package report

import (
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	report2 "atlas-channel/kafka/message/report"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	socketwriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestResultPacketMapping proves the pure (kind, status, errorCode) ->
// (writerName, ok) mapping (design.md §4.5, v1 result policy) without
// needing a session, server, or writer.Producer.
func TestResultPacketMapping(t *testing.T) {
	cases := []struct {
		name       string
		event      report2.StatusEvent
		wantWriter string
		wantOk     bool
	}{
		{"sue created", report2.StatusEvent{Kind: report2.KindSue, Status: report2.EventStatusCreated}, reportcb.SueCharacterResultWriter, true},
		{"sue not found", report2.StatusEvent{Kind: report2.KindSue, Status: report2.EventStatusError, ErrorCode: report2.ErrorCodeNotFound}, reportcb.SueCharacterResultWriter, true},
		{"sue internal", report2.StatusEvent{Kind: report2.KindSue, Status: report2.EventStatusError, ErrorCode: report2.ErrorCodeInternal}, reportcb.SueCharacterResultWriter, true},
		{"claim created", report2.StatusEvent{Kind: report2.KindClaim, Status: report2.EventStatusCreated}, reportcb.ClaimResultWriter, true},
		{"claim not found", report2.StatusEvent{Kind: report2.KindClaim, Status: report2.EventStatusError, ErrorCode: report2.ErrorCodeNotFound}, reportcb.ClaimResultWriter, true},
		{"claim internal", report2.StatusEvent{Kind: report2.KindClaim, Status: report2.EventStatusError, ErrorCode: report2.ErrorCodeInternal}, reportcb.ClaimResultWriter, true},
		{"unknown kind dropped", report2.StatusEvent{Kind: "bogus", Status: report2.EventStatusCreated}, "", false},
		{"unknown status dropped", report2.StatusEvent{Kind: report2.KindSue, Status: "PENDING"}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writerName, body, ok := resultPacket(tc.event)
			if ok != tc.wantOk {
				t.Fatalf("ok: got %v want %v", ok, tc.wantOk)
			}
			if !ok {
				return
			}
			if writerName != tc.wantWriter {
				t.Errorf("writer: got %s want %s", writerName, tc.wantWriter)
			}
			if body == nil {
				t.Error("expected non-nil body")
			}
		})
	}
}

// --- dispatch tests: recording-seam pattern (mirrors kafka/consumer/rps and
// kafka/consumer/mount) ---

// announceCall captures one invocation of the reportAnnouncer seam: which
// character's session it targeted, the writer selected, and the wire-encoded
// bytes produced by that writer's body func against a fixed operations table.
type announceCall struct {
	characterId uint32
	writerName  string
	bytes       []byte
}

// sueOperations and claimOperations mirror the per-writer operations tables
// each writer's result/mode byte is resolved from (WithResolvedCode,
// libs/atlas-packet/resolve.go). Each real tenant WriterConfig entry carries
// its own Options bag, so SueCharacterResult's SUCCESS=0 and
// ClaimResult's SUCCESS=2 never collide in production - these are two
// separate maps to mirror that. Values are arbitrary but distinct per key,
// only used to prove the correct code was selected.
var sueOperations = map[string]interface{}{
	string(writer.SueResultSuccess):        float64(0),
	string(writer.SueResultUnableToLocate): float64(1),
	string(writer.SueResultGenericFailure): float64(4),
}

var claimOperations = map[string]interface{}{
	string(writer.ClaimResultSuccessCode):    float64(2),
	string(writer.ClaimResultTryAgain):       float64(0x47),
	string(writer.ClaimResultRecheckName):    float64(0x48),
	string(writer.ClaimResultExceeded):       float64(0x49),
	string(writer.ClaimResultNotEnoughMesos): float64(0x4A),
}

// withRecordingAnnouncer swaps the package-level reportAnnouncer seam for a
// recording stub that immediately invokes the passed body encoder (with the
// operations table matching the selected writer) and records the
// characterId + writer name + resulting bytes.
func withRecordingAnnouncer(t *testing.T) (restore func(), calls *[]announceCall) {
	t.Helper()
	var recorded []announceCall
	orig := reportAnnouncer
	l, _ := testlog.NewNullLogger()
	ctx := context.Background()
	reportAnnouncer = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ server.Model, characterId uint32, writerName string, body packet.Encode) {
		ops := sueOperations
		if writerName == reportcb.ClaimResultWriter {
			ops = claimOperations
		}
		b := body(l, ctx)(map[string]interface{}{"operations": ops})
		recorded = append(recorded, announceCall{characterId: characterId, writerName: writerName, bytes: b})
	}
	return func() { reportAnnouncer = orig }, &recorded
}

// withRecordingFeeCharger swaps the claim-fee seam for a recorder, so tests
// assert which character was charged without a live session registry or a
// Kafka producer.
func withRecordingFeeCharger(t *testing.T) (restore func(), charged *[]uint32) {
	t.Helper()
	var recorded []uint32
	orig := claimFeeCharger
	claimFeeCharger = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, characterId uint32) {
		recorded = append(recorded, characterId)
	}
	return func() { claimFeeCharger = orig }, &recorded
}

// nullLogger returns a discarding logrus.Logger backed by testlog's null
// hook, for call sites that need a *logrus.Logger but don't assert on log
// output. Using it instead of bare logrus.New() avoids stray warn-level
// stderr noise during `go test` (e.g. TestHandleStatusEvent_UnmappedCombo_
// DoesNothing's unmapped-combo path logs at Warn).
func nullLogger() *logrus.Logger {
	l, _ := testlog.NewNullLogger()
	return l
}

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
	ch := channelconst.NewModel(0, 1)
	return server.NewProcessor(nullLogger(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

// newZeroFieldTestServer registers a server whose world/channel (0, 0) match
// session.NewSession's un-set default field. session.Processor exposes no
// public setter for a session's worldId/channelId (only the production
// Create(ch, locale) path sets them, and it also writes a hello packet to a
// live net.Conn - unusable in a unit test); the real-session tests below use
// this server instead of newTestServer so IfPresentByCharacterId's
// world/channel filters (session/processor.go ByCharacterIdModelProvider)
// actually match the directly-registered test session.
func newZeroFieldTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channelconst.NewModel(0, 0)
	return server.NewProcessor(nullLogger(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

func decodeSueResult(t *testing.T, b []byte) reportcb.SueCharacterResult {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	req := request.Request(b)
	reader := request.NewRequestReader(&req, 0)
	var m reportcb.SueCharacterResult
	m.Decode(l, context.Background())(&reader, nil)
	return m
}

func decodeClaimSuccess(t *testing.T, b []byte) reportcb.ClaimResultSuccess {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	req := request.Request(b)
	reader := request.NewRequestReader(&req, 0)
	var m reportcb.ClaimResultSuccess
	m.Decode(l, context.Background())(&reader, nil)
	return m
}

func decodeClaimNotice(t *testing.T, b []byte) reportcb.ClaimResultNotice {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	req := request.Request(b)
	reader := request.NewRequestReader(&req, 0)
	var m reportcb.ClaimResultNotice
	m.Decode(l, context.Background())(&reader, nil)
	return m
}

// TestHandleStatusEvent_SueCreated_AnnouncesSuccess asserts a sue CREATED
// event selects SueCharacterResult SUCCESS and targets the reporter's
// session.
func TestHandleStatusEvent_SueCreated_AnnouncesSuccess(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleStatusEvent(sc, nil)
	h(nullLogger(), ctx, report2.StatusEvent{
		Kind:       report2.KindSue,
		WorldId:    sc.WorldId(),
		ReporterId: 4001,
		Status:     report2.EventStatusCreated,
	})

	if len(*calls) != 1 {
		t.Fatalf("want 1 announce call, got %d", len(*calls))
	}
	call := (*calls)[0]
	if call.characterId != 4001 {
		t.Fatalf("want session targeted for character 4001, got %d", call.characterId)
	}
	if call.writerName != reportcb.SueCharacterResultWriter {
		t.Fatalf("want writer %s, got %s", reportcb.SueCharacterResultWriter, call.writerName)
	}
	res := decodeSueResult(t, call.bytes)
	if res.Result() != 0 {
		t.Fatalf("want resolved SUCCESS code 0, got %d", res.Result())
	}
}

// TestHandleStatusEvent_SueErrorNotFound_AnnouncesUnableToLocate asserts a
// sue ERROR/NOT_FOUND event maps to UNABLE_TO_LOCATE, distinct from the
// INTERNAL mapping below (the required not-collapsed distinction).
func TestHandleStatusEvent_SueErrorNotFound_AnnouncesUnableToLocate(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleStatusEvent(sc, nil)
	h(nullLogger(), ctx, report2.StatusEvent{
		Kind:       report2.KindSue,
		WorldId:    sc.WorldId(),
		ReporterId: 4002,
		Status:     report2.EventStatusError,
		ErrorCode:  report2.ErrorCodeNotFound,
	})

	res := decodeSueResult(t, (*calls)[0].bytes)
	if res.Result() != 1 {
		t.Fatalf("want resolved UNABLE_TO_LOCATE code 1, got %d", res.Result())
	}
}

// TestHandleStatusEvent_SueErrorInternal_AnnouncesGenericFailure asserts a
// sue ERROR/INTERNAL event maps to GENERIC_FAILURE - a DIFFERENT code than
// NOT_FOUND above, proving the two error paths are not collapsed.
func TestHandleStatusEvent_SueErrorInternal_AnnouncesGenericFailure(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleStatusEvent(sc, nil)
	h(nullLogger(), ctx, report2.StatusEvent{
		Kind:       report2.KindSue,
		WorldId:    sc.WorldId(),
		ReporterId: 4003,
		Status:     report2.EventStatusError,
		ErrorCode:  report2.ErrorCodeInternal,
	})

	res := decodeSueResult(t, (*calls)[0].bytes)
	if res.Result() != 4 {
		t.Fatalf("want resolved GENERIC_FAILURE code 4, got %d", res.Result())
	}
	if res.Result() == 1 {
		t.Fatalf("INTERNAL must not collapse into the NOT_FOUND/UNABLE_TO_LOCATE code")
	}
}

// TestHandleStatusEvent_ClaimCreated_AnnouncesSuccessWithRemaining asserts a
// claim CREATED event selects ClaimResultSuccess carrying the quota standing
// the event supplied — not a constant. A wrong value here is exactly the bug
// play-testing surfaced: every claim reported "100 left".
func TestHandleStatusEvent_ClaimCreated_AnnouncesSuccessWithRemaining(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()
	restoreFee, _ := withRecordingFeeCharger(t)
	defer restoreFee()

	h := handleStatusEvent(sc, nil)
	h(nullLogger(), ctx, report2.StatusEvent{
		Kind:         report2.KindClaim,
		WorldId:      sc.WorldId(),
		ReporterId:   4004,
		Status:       report2.EventStatusCreated,
		HasRemaining: true,
		Remaining:    97,
	})

	if (*calls)[0].writerName != reportcb.ClaimResultWriter {
		t.Fatalf("want writer %s, got %s", reportcb.ClaimResultWriter, (*calls)[0].writerName)
	}
	res := decodeClaimSuccess(t, (*calls)[0].bytes)
	if res.Mode() != 2 {
		t.Fatalf("want resolved SUCCESS mode 2, got %d", res.Mode())
	}
	if !res.HasRemaining() {
		t.Fatalf("want hasRemaining=true")
	}
	if res.Remaining() != 97 {
		t.Fatalf("want remaining=97 from the event, got %d", res.Remaining())
	}
}

// TestHandleStatusEvent_ClaimQuotaExceeded_AnnouncesExceeded asserts the
// quota rejection maps to the client's dedicated EXCEEDED notice rather than
// collapsing into the generic TRY_AGAIN.
func TestHandleStatusEvent_ClaimQuotaExceeded_AnnouncesExceeded(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()
	restoreFee, charged := withRecordingFeeCharger(t)
	defer restoreFee()

	h := handleStatusEvent(sc, nil)
	h(nullLogger(), ctx, report2.StatusEvent{
		Kind:       report2.KindClaim,
		WorldId:    sc.WorldId(),
		ReporterId: 4009,
		Status:     report2.EventStatusError,
		ErrorCode:  report2.ErrorCodeQuotaExceeded,
	})

	res := decodeClaimNotice(t, (*calls)[0].bytes)
	if res.Mode() != 0x49 {
		t.Fatalf("want resolved EXCEEDED mode 0x49 (fixture value), got %#x", res.Mode())
	}
	if len(*charged) != 0 {
		t.Fatalf("a rejected claim must not be charged, got %v", *charged)
	}
}

// TestHandleStatusEvent_ClaimCreated_ChargesFee asserts the fee is charged on
// the created path, and only there.
func TestHandleStatusEvent_ClaimCreated_ChargesFee(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, _ := withRecordingAnnouncer(t)
	defer restore()
	restoreFee, charged := withRecordingFeeCharger(t)
	defer restoreFee()

	h := handleStatusEvent(sc, nil)
	h(nullLogger(), ctx, report2.StatusEvent{
		Kind:         report2.KindClaim,
		WorldId:      sc.WorldId(),
		ReporterId:   4010,
		Status:       report2.EventStatusCreated,
		HasRemaining: true,
		Remaining:    12,
	})
	// A sue is free — no meso mode exists on SUE_CHARACTER_RESULT.
	h(nullLogger(), ctx, report2.StatusEvent{
		Kind:       report2.KindSue,
		WorldId:    sc.WorldId(),
		ReporterId: 4011,
		Status:     report2.EventStatusCreated,
	})

	if len(*charged) != 1 || (*charged)[0] != 4010 {
		t.Fatalf("want exactly character 4010 charged, got %v", *charged)
	}
}

// TestHandleStatusEvent_ClaimErrorNotFound_AnnouncesRecheckName asserts a
// claim ERROR/NOT_FOUND event maps to RECHECK_NAME.
func TestHandleStatusEvent_ClaimErrorNotFound_AnnouncesRecheckName(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleStatusEvent(sc, nil)
	h(nullLogger(), ctx, report2.StatusEvent{
		Kind:       report2.KindClaim,
		WorldId:    sc.WorldId(),
		ReporterId: 4005,
		Status:     report2.EventStatusError,
		ErrorCode:  report2.ErrorCodeNotFound,
	})

	res := decodeClaimNotice(t, (*calls)[0].bytes)
	if res.Mode() != 0x48 {
		t.Fatalf("want resolved RECHECK_NAME mode 0x48, got %#x", res.Mode())
	}
}

// TestHandleStatusEvent_ClaimErrorInternal_AnnouncesTryAgain asserts a claim
// ERROR/INTERNAL event maps to TRY_AGAIN - a DIFFERENT mode than
// RECHECK_NAME above, proving the two error paths are not collapsed.
func TestHandleStatusEvent_ClaimErrorInternal_AnnouncesTryAgain(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleStatusEvent(sc, nil)
	h(nullLogger(), ctx, report2.StatusEvent{
		Kind:       report2.KindClaim,
		WorldId:    sc.WorldId(),
		ReporterId: 4006,
		Status:     report2.EventStatusError,
		ErrorCode:  report2.ErrorCodeInternal,
	})

	res := decodeClaimNotice(t, (*calls)[0].bytes)
	if res.Mode() != 0x47 {
		t.Fatalf("want resolved TRY_AGAIN mode 0x47, got %#x", res.Mode())
	}
	if res.Mode() == 0x48 {
		t.Fatalf("INTERNAL must not collapse into the NOT_FOUND/RECHECK_NAME mode")
	}
}

// TestHandleStatusEvent_WrongWorld_DoesNothing guards the world gate
// (sc.IsWorld).
func TestHandleStatusEvent_WrongWorld_DoesNothing(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleStatusEvent(sc, nil)
	h(nullLogger(), ctx, report2.StatusEvent{
		Kind:       report2.KindSue,
		WorldId:    sc.WorldId() + 1,
		ReporterId: 4007,
		Status:     report2.EventStatusCreated,
	})

	if len(*calls) != 0 {
		t.Fatalf("wrong world: want no effects, got %d", len(*calls))
	}
}

// TestHandleStatusEvent_UnmappedCombo_DoesNothing guards against an
// unmapped kind/status combination reaching the announcer.
func TestHandleStatusEvent_UnmappedCombo_DoesNothing(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleStatusEvent(sc, nil)
	h(nullLogger(), ctx, report2.StatusEvent{
		Kind:       "bogus",
		WorldId:    sc.WorldId(),
		ReporterId: 4008,
		Status:     report2.EventStatusCreated,
	})

	if len(*calls) != 0 {
		t.Fatalf("unmapped combo: want no effects, got %d", len(*calls))
	}
}

// --- writer-not-found skip path: exercises the REAL reportAnnouncer (not
// the recording seam) end to end through a real registered session, proving
// the load-bearing gap (task brief) is closed at this call site: a tenant
// whose config lacks the mapped writer must skip delivery at debug level,
// never propagate an error. ---

func TestHandleStatusEvent_WriterNotFound_SkipsWithoutError(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newZeroFieldTestServer(t, tm)

	sessionId := uuid.New()
	characterId := uint32(5001)
	s := session.NewSession(sessionId, tm, 0, nil)
	session.AddSessionToRegistry(tm.Id(), s)
	defer session.ClearRegistryForTenant(tm.Id())
	_ = session.NewProcessor(nullLogger(), ctx).SetCharacterId(sessionId, characterId)

	// These two tests exercise the real reportAnnouncer, but the fee charger
	// is still stubbed: it is a separate concern, and its real form reaches
	// for a Kafka producer this test has no broker for.
	restoreFee, _ := withRecordingFeeCharger(t)
	defer restoreFee()

	logger, hook := testlog.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	// A writer.Producer whose tenant config maps NO report writer at all -
	// exactly what writer.ProducerGetter returns for a writer BuildWriterProducer
	// dropped at boot (libs/atlas-opcodes/producer.go / libs/atlas-socket/writer/writer.go).
	notFound := errors.New("writer not found")
	wp := writer.Producer(func(_ string) (socketwriter.BodyFunc, error) {
		return nil, notFound
	})

	h := handleStatusEvent(sc, wp)
	h(logger, ctx, report2.StatusEvent{
		Kind:       report2.KindClaim,
		WorldId:    sc.WorldId(),
		ReporterId: characterId,
		Status:     report2.EventStatusCreated,
	})

	for _, e := range hook.AllEntries() {
		if e.Level <= logrus.WarnLevel {
			t.Fatalf("writer-not-found must not error/warn (config presence is the feature gate); got %s-level entry: %q", e.Level, e.Message)
		}
	}

	foundDebugSkip := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.DebugLevel {
			foundDebugSkip = true
		}
	}
	if !foundDebugSkip {
		t.Fatal("want a debug-level skip log entry when the tenant has no mapped writer, got none")
	}
}

// TestHandleStatusEvent_WriterFound_DeliversToRealConnection is the control
// case for the skip-path test above: with a writer.Producer that DOES
// resolve the writer and a session backed by a real net.Conn (net.Pipe), the
// real reportAnnouncer proceeds through session.Announce and actually writes
// bytes to the connection - proving the writer-not-found test above is
// exercising a genuine short-circuit, not a session lookup that always
// fails/no-ops regardless of the writer check.
func TestHandleStatusEvent_WriterFound_DeliversToRealConnection(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newZeroFieldTestServer(t, tm)

	serverConn, clientConn := net.Pipe()
	defer func() { _ = serverConn.Close() }()
	defer func() { _ = clientConn.Close() }()

	sessionId := uuid.New()
	characterId := uint32(5003)
	s := session.NewSession(sessionId, tm, 0, serverConn)
	session.AddSessionToRegistry(tm.Id(), s)
	defer session.ClearRegistryForTenant(tm.Id())
	_ = session.NewProcessor(nullLogger(), ctx).SetCharacterId(sessionId, characterId)

	restoreFee, _ := withRecordingFeeCharger(t)
	defer restoreFee()

	logger, hook := testlog.NewNullLogger()

	wp := writer.Producer(func(_ string) (socketwriter.BodyFunc, error) {
		return socketwriter.BodyFunc(func(_ logrus.FieldLogger, _ context.Context) func(packet.Encode) []byte {
			return func(_ packet.Encode) []byte { return []byte{0x01, 0x02, 0x03} }
		}), nil
	})

	readErr := make(chan error, 1)
	readN := make(chan int, 1)
	go func() {
		buf := make([]byte, 64)
		n, err := clientConn.Read(buf)
		readN <- n
		readErr <- err
	}()

	h := handleStatusEvent(sc, wp)
	h(logger, ctx, report2.StatusEvent{
		Kind:       report2.KindClaim,
		WorldId:    sc.WorldId(),
		ReporterId: characterId,
		Status:     report2.EventStatusCreated,
	})

	select {
	case n := <-readN:
		if err := <-readErr; err != nil {
			t.Fatalf("read from session connection: %v", err)
		}
		if n == 0 {
			t.Fatal("want bytes written to the real session connection, got none")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a write to the session connection - delivery did not reach announceEncrypted")
	}

	for _, e := range hook.AllEntries() {
		if e.Level <= logrus.WarnLevel {
			t.Fatalf("writer-found delivery must not error/warn; got %s-level entry: %q", e.Level, e.Message)
		}
	}
}
