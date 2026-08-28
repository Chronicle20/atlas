package socket

import (
	"atlas-channel/configuration"
	"atlas-channel/configuration/tenant"
	"atlas-channel/configuration/tenant/diagnostics"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	tenant2 "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestNewPacketTracer(t *testing.T) {
	tm, err := tenant2.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}

	l, h := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	names := map[uint16]string{0x0001: "USER_CHAT"}
	tracer := NewPacketTracer(l, tm, names)

	// Phase 1: flag off -- nothing is formatted (FR-2.1).
	tracer(uuid.New(), 0x0001, []byte{0x01, 0x00, 0xaa})
	if got := len(h.AllEntries()); got != 0 {
		t.Fatalf("phase 1 (flag off): expected 0 entries, got %d", got)
	}
	h.Reset()

	// Phase 2: flag on, known handler.
	configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
		tm.Id(): {Diagnostics: diagnostics.RestModel{TracePackets: true}},
	})
	tracer(uuid.New(), 0x0001, []byte{0x01, 0x00, 0xaa})
	entries := h.AllEntries()
	if len(entries) != 1 {
		t.Fatalf("phase 2 (flag on, known handler): expected 1 entry, got %d", len(entries))
	}
	if entries[0].Level != logrus.DebugLevel {
		t.Fatalf("phase 2: expected DebugLevel, got %v", entries[0].Level)
	}
	msg := entries[0].Message
	if !strings.HasPrefix(msg, "[PKT IN ] handler=USER_CHAT op=0x0001 len=3 session=") {
		t.Fatalf("phase 2: unexpected message prefix: %q", msg)
	}
	if !strings.Contains(msg, "\n0000  01 00 aa") {
		t.Fatalf("phase 2: expected hex dump line, got: %q", msg)
	}
	h.Reset()

	// Phase 3: flag on, unknown opcode (FR-3.4).
	tracer(uuid.New(), 0x00ff, []byte{0xff, 0x00})
	entries = h.AllEntries()
	if len(entries) != 1 {
		t.Fatalf("phase 3 (unknown opcode): expected 1 entry, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Message, "handler=<none> op=0x00ff len=2") {
		t.Fatalf("phase 3: unexpected message: %q", entries[0].Message)
	}
	h.Reset()

	// Phase 4: flag on, pod at Info -- suppressed (FR-2.2).
	l.SetLevel(logrus.InfoLevel)
	tracer(uuid.New(), 0x0001, []byte{0x01, 0x00, 0xaa})
	if got := len(h.AllEntries()); got != 0 {
		t.Fatalf("phase 4 (pod at Info): expected 0 entries, got %d", got)
	}
	h.Reset()
	l.SetLevel(logrus.DebugLevel)

	// Phase 5: other tenant's flag does not leak to this tenant (FR-2.6).
	otherTenantId := uuid.New()
	configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
		otherTenantId: {Diagnostics: diagnostics.RestModel{TracePackets: true}},
	})
	tracer(uuid.New(), 0x0001, []byte{0x01, 0x00, 0xaa})
	if got := len(h.AllEntries()); got != 0 {
		t.Fatalf("phase 5 (other tenant's flag): expected 0 entries, got %d", got)
	}
	h.Reset()
}
