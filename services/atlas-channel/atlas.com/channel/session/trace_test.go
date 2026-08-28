package session

import (
	"atlas-channel/configuration"
	"atlas-channel/configuration/tenant"
	"atlas-channel/configuration/tenant/diagnostics"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"

	tenant2 "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestTracePacketOut_ShortPayloadRendersNoOpcode is in package session
// (rather than session_test) because tracePacketOut is unexported.
func TestTracePacketOut_ShortPayloadRendersNoOpcode(t *testing.T) {
	tm, err := tenant2.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}

	configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
		tm.Id(): {Diagnostics: diagnostics.RestModel{TracePackets: true}},
	})

	l, h := logtest.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	tracePacketOut(l, tm, "CHARACTER_DATA", uuid.New(), []byte{0x7d})

	entries := h.AllEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Message, "op=n/a len=1") {
		t.Fatalf("message = %q, want it to contain %q", entries[0].Message, "op=n/a len=1")
	}
}
