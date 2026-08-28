package session

import (
	"atlas-login/configuration"
	"atlas-login/configuration/tenant"
	"atlas-login/configuration/tenant/diagnostics"
	"atlas-login/socket/writer"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"

	socketpacket "github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant2 "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestAnnounce_TracesOutboundPacket exercises the trace gate end to end
// through Announce: flag off emits nothing (FR-2.1), flag on emits exactly
// one [PKT OUT] entry rendering the writer's plaintext bytes before
// encryption (FR-4.2, FR-4.4), and a writer-resolution failure still
// returns the original error with no trace emitted (FR-4.5).
func TestAnnounce_TracesOutboundPacket(t *testing.T) {
	tm, err := tenant2.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}
	ctx := tenant2.WithContext(context.Background(), tm)
	sessionId := uuid.New()

	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	})
	go func() { _, _ = io.Copy(io.Discard, clientConn) }()

	s := NewSession(sessionId, tm, 8, serverConn)

	l, h := logtest.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	body := []byte{0x7d, 0x00, 0xaa, 0xbb}
	encoder := socketpacket.Encode(func(_ logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
		return func(_ map[string]interface{}) []byte { return nil }
	})

	// Phase 1: flag off -- no trace entry.
	wp := writer.Producer(func(_ string) (writer.BodyFunc, error) {
		return func(_ logrus.FieldLogger, _ context.Context) func(encoder socketpacket.Encode) []byte {
			return func(_ socketpacket.Encode) []byte { return body }
		}, nil
	})
	op := Announce(l)(ctx)(wp)("LOGIN_RESULT")(encoder)
	if err := op(s); err != nil {
		t.Fatalf("phase 1 (flag off): unexpected error: %v", err)
	}
	for _, e := range h.AllEntries() {
		if strings.HasPrefix(e.Message, "[PKT OUT]") {
			t.Fatalf("phase 1 (flag off): unexpected [PKT OUT] entry: %q", e.Message)
		}
	}
	h.Reset()

	// Phase 2: flag on -- exactly one [PKT OUT] entry with the writer's
	// plaintext bytes.
	configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
		tm.Id(): {Diagnostics: diagnostics.RestModel{TracePackets: true}},
	})
	if err := op(s); err != nil {
		t.Fatalf("phase 2 (flag on): unexpected error: %v", err)
	}
	entries := h.AllEntries()
	if len(entries) != 1 {
		t.Fatalf("phase 2 (flag on): expected 1 entry, got %d", len(entries))
	}
	if entries[0].Level != logrus.DebugLevel {
		t.Fatalf("phase 2 (flag on): expected DebugLevel, got %v", entries[0].Level)
	}
	wantPrefix := "[PKT OUT] writer=LOGIN_RESULT op=0x007d len=4 session=" + sessionId.String()
	if !strings.HasPrefix(entries[0].Message, wantPrefix) {
		t.Fatalf("phase 2 (flag on): message = %q, want prefix %q", entries[0].Message, wantPrefix)
	}
	wantBody := "0000  7d 00 aa bb                                       |}...|"
	if !strings.Contains(entries[0].Message, wantBody) {
		t.Fatalf("phase 2 (flag on): message = %q, want it to contain %q", entries[0].Message, wantBody)
	}
	h.Reset()

	// Phase 3: writer resolution fails -- Announce returns the error
	// unchanged and no trace entry is emitted (FR-4.5).
	wantErr := errors.New("writer not found")
	failingWp := writer.Producer(func(_ string) (writer.BodyFunc, error) {
		return nil, wantErr
	})
	failingOp := Announce(l)(ctx)(failingWp)("LOGIN_RESULT")(encoder)
	if err := failingOp(s); !errors.Is(err, wantErr) {
		t.Fatalf("phase 3 (writer resolution fails): err = %v, want %v", err, wantErr)
	}
	for _, e := range h.AllEntries() {
		if strings.HasPrefix(e.Message, "[PKT OUT]") {
			t.Fatalf("phase 3 (writer resolution fails): unexpected [PKT OUT] entry: %q", e.Message)
		}
	}
}

// TestTracePacketOut_HelloRendersNoOpcode is in package session (rather
// than session_test) because tracePacketOut is unexported.
func TestTracePacketOut_HelloRendersNoOpcode(t *testing.T) {
	tm, err := tenant2.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}

	configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
		tm.Id(): {Diagnostics: diagnostics.RestModel{TracePackets: true}},
	})

	l, h := logtest.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	tracePacketOut(l, tm, "<hello>", uuid.New(), []byte{0x0e, 0x00, 0x53, 0x00})

	entries := h.AllEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !strings.HasPrefix(entries[0].Message, "[PKT OUT] writer=<hello> op=n/a len=4") {
		t.Fatalf("message = %q, want it to start with %q", entries[0].Message, "[PKT OUT] writer=<hello> op=n/a len=4")
	}
}
