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
	names := map[uint16]string{0x0001: "USER_CHAT"}

	t.Run("flag off -- nothing is formatted (FR-2.1)", func(t *testing.T) {
		tm, err := tenant2.Register(uuid.New(), "GMS", 83, 1)
		if err != nil {
			t.Fatalf("failed to register tenant: %v", err)
		}

		l, h := test.NewNullLogger()
		l.SetLevel(logrus.DebugLevel)
		tracer := NewPacketTracer(l, tm, names)

		tracer(uuid.New(), 0x0001, []byte{0x01, 0x00, 0xaa})
		if got := len(h.AllEntries()); got != 0 {
			t.Fatalf("expected 0 entries, got %d", got)
		}
	})

	t.Run("flag on, known handler", func(t *testing.T) {
		tm, err := tenant2.Register(uuid.New(), "GMS", 83, 1)
		if err != nil {
			t.Fatalf("failed to register tenant: %v", err)
		}

		l, h := test.NewNullLogger()
		l.SetLevel(logrus.DebugLevel)
		tracer := NewPacketTracer(l, tm, names)

		configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
			tm.Id(): {Diagnostics: diagnostics.RestModel{TracePackets: true}},
		})

		tracer(uuid.New(), 0x0001, []byte{0x01, 0x00, 0xaa})
		entries := h.AllEntries()
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Level != logrus.DebugLevel {
			t.Fatalf("expected DebugLevel, got %v", entries[0].Level)
		}
		msg := entries[0].Message
		if !strings.HasPrefix(msg, "[PKT IN ] handler=USER_CHAT op=0x0001 len=3 session=") {
			t.Fatalf("unexpected message prefix: %q", msg)
		}
		if !strings.Contains(msg, "\n0000  01 00 aa") {
			t.Fatalf("expected hex dump line, got: %q", msg)
		}
	})

	t.Run("flag on, unknown opcode (FR-3.4)", func(t *testing.T) {
		tm, err := tenant2.Register(uuid.New(), "GMS", 83, 1)
		if err != nil {
			t.Fatalf("failed to register tenant: %v", err)
		}

		l, h := test.NewNullLogger()
		l.SetLevel(logrus.DebugLevel)
		tracer := NewPacketTracer(l, tm, names)

		configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
			tm.Id(): {Diagnostics: diagnostics.RestModel{TracePackets: true}},
		})

		tracer(uuid.New(), 0x00ff, []byte{0xff, 0x00})
		entries := h.AllEntries()
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if !strings.Contains(entries[0].Message, "handler=<none> op=0x00ff len=2") {
			t.Fatalf("unexpected message: %q", entries[0].Message)
		}
	})

	t.Run("flag on, pod at Info -- suppressed (FR-2.2)", func(t *testing.T) {
		tm, err := tenant2.Register(uuid.New(), "GMS", 83, 1)
		if err != nil {
			t.Fatalf("failed to register tenant: %v", err)
		}

		l, h := test.NewNullLogger()
		l.SetLevel(logrus.InfoLevel)
		tracer := NewPacketTracer(l, tm, names)

		configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
			tm.Id(): {Diagnostics: diagnostics.RestModel{TracePackets: true}},
		})

		tracer(uuid.New(), 0x0001, []byte{0x01, 0x00, 0xaa})
		if got := len(h.AllEntries()); got != 0 {
			t.Fatalf("expected 0 entries, got %d", got)
		}
	})

	t.Run("other tenant's flag does not leak to this tenant (FR-2.6)", func(t *testing.T) {
		tm, err := tenant2.Register(uuid.New(), "GMS", 83, 1)
		if err != nil {
			t.Fatalf("failed to register tenant: %v", err)
		}

		l, h := test.NewNullLogger()
		l.SetLevel(logrus.DebugLevel)
		tracer := NewPacketTracer(l, tm, names)

		otherTenantId := uuid.New()
		configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
			otherTenantId: {Diagnostics: diagnostics.RestModel{TracePackets: true}},
		})

		tracer(uuid.New(), 0x0001, []byte{0x01, 0x00, 0xaa})
		if got := len(h.AllEntries()); got != 0 {
			t.Fatalf("expected 0 entries, got %d", got)
		}
	})
}
