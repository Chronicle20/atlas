package serverbound

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func newTestLogCtx() (logrus.FieldLogger, context.Context) {
	l, _ := testlog.NewNullLogger()
	return l, context.Background()
}

// TestDueyActionDecode exercises the DUEY_ACTION serverbound arms directly
// against hand-built wire bytes (design.md §5.4) rather than round-tripping
// through Encode, so the asymmetric SEND body (quick vs NPC) and the
// non-consumption of trailing garbage on the NPC path are both provable.
func TestDueyActionDecode(t *testing.T) {
	l, ctx := newTestLogCtx()

	t.Run("mode send", func(t *testing.T) {
		body := sendNpcBytes()
		buf := append([]byte{0x02}, body...)
		r := newReader(buf)
		var a Action
		a.Decode(l, ctx)(&r, nil)
		if a.Mode() != 2 {
			t.Errorf("Mode() = %d, want 2", a.Mode())
		}
	})

	t.Run("send npc", func(t *testing.T) {
		buf := sendNpcBytes()
		r := newReader(buf)
		var a ActionSend
		a.Decode(l, ctx)(&r, nil)
		if a.InventoryType() != 1 {
			t.Errorf("InventoryType() = %d, want 1", a.InventoryType())
		}
		if a.Slot() != 5 {
			t.Errorf("Slot() = %d, want 5", a.Slot())
		}
		if a.Quantity() != 1 {
			t.Errorf("Quantity() = %d, want 1", a.Quantity())
		}
		if a.Mesos() != 1000 {
			t.Errorf("Mesos() = %d, want 1000", a.Mesos())
		}
		if a.RecipientName() != "Bob" {
			t.Errorf("RecipientName() = %q, want %q", a.RecipientName(), "Bob")
		}
		if a.Quick() != false {
			t.Errorf("Quick() = %t, want false", a.Quick())
		}
		if a.Message() != "" {
			t.Errorf("Message() = %q, want empty", a.Message())
		}
		if a.TicketRef() != 0 {
			t.Errorf("TicketRef() = %d, want 0", a.TicketRef())
		}
		if r.Available() != 0 {
			t.Errorf("reader has %d unconsumed bytes, want 0", r.Available())
		}
	})

	t.Run("send quick", func(t *testing.T) {
		buf := sendQuickBytes()
		r := newReader(buf)
		var a ActionSend
		a.Decode(l, ctx)(&r, nil)
		if a.Quick() != true {
			t.Errorf("Quick() = %t, want true", a.Quick())
		}
		if a.Message() != "hi" {
			t.Errorf("Message() = %q, want %q", a.Message(), "hi")
		}
		if a.TicketRef() != 1000000 {
			t.Errorf("TicketRef() = %d, want 1000000", a.TicketRef())
		}
		if r.Available() != 0 {
			t.Errorf("reader has %d unconsumed bytes, want 0", r.Available())
		}
	})

	t.Run("send npc trailing garbage", func(t *testing.T) {
		buf := append(sendNpcBytes(), 0xDE, 0xAD, 0xBE, 0xEF)
		r := newReader(buf)
		var a ActionSend
		a.Decode(l, ctx)(&r, nil)
		if r.Available() != 4 {
			t.Errorf("reader has %d remaining bytes, want 4 (trailing garbage must not be consumed)", r.Available())
		}
	})

	t.Run("receive", func(t *testing.T) {
		buf := []byte{0x07, 0x00, 0x00, 0x00}
		r := newReader(buf)
		var a ActionReceive
		a.Decode(l, ctx)(&r, nil)
		if a.ParcelId() != 7 {
			t.Errorf("ParcelId() = %d, want 7", a.ParcelId())
		}
		if r.Available() != 0 {
			t.Errorf("reader has %d unconsumed bytes, want 0", r.Available())
		}
	})

	t.Run("discard", func(t *testing.T) {
		buf := []byte{0x07, 0x00, 0x00, 0x00}
		r := newReader(buf)
		var a ActionDiscard
		a.Decode(l, ctx)(&r, nil)
		if a.ParcelId() != 7 {
			t.Errorf("ParcelId() = %d, want 7", a.ParcelId())
		}
		if r.Available() != 0 {
			t.Errorf("reader has %d unconsumed bytes, want 0", r.Available())
		}
	})

	t.Run("close", func(t *testing.T) {
		r := newReader([]byte{})
		var a ActionClose
		a.Decode(l, ctx)(&r, nil)
		if r.Available() != 0 {
			t.Errorf("reader has %d unconsumed bytes, want 0", r.Available())
		}
	})
}

// sendNpcBytes is the NPC-path SEND body (design.md §5.4, v83
// CTabSend::SendParcel @0x6F36A8): the writer stops after the quick flag.
func sendNpcBytes() []byte {
	buf := []byte{
		0x01,       // invType
		0x05, 0x00, // slot (uint16 LE)
		0x01, 0x00, // quantity (uint16 LE)
		0xE8, 0x03, 0x00, 0x00, // mesos (uint32 LE) = 1000
	}
	buf = append(buf, asciiString("Bob")...)
	buf = append(buf, 0x00) // quick = false
	return buf
}

// sendQuickBytes is the quick-send SEND body (design.md §5.4, v83
// CTabSend::SendParcel @0x6F1DF5): message + ticketRef follow the flag.
func sendQuickBytes() []byte {
	buf := []byte{
		0x01,
		0x05, 0x00,
		0x01, 0x00,
		0xE8, 0x03, 0x00, 0x00,
	}
	buf = append(buf, asciiString("Bob")...)
	buf = append(buf, 0x01) // quick = true
	buf = append(buf, asciiString("hi")...)
	buf = append(buf, 0x40, 0x42, 0x0F, 0x00) // ticketRef (uint32 LE) = 1000000
	return buf
}

// asciiString mirrors response.Writer.WriteAsciiString for a plain-ASCII
// input: a uint16 LE length prefix followed by the raw bytes.
func asciiString(s string) []byte {
	buf := []byte{byte(len(s)), byte(len(s) >> 8)}
	return append(buf, []byte(s)...)
}

func newReader(buf []byte) request.Reader {
	req := request.Request(buf)
	return request.NewRequestReader(&req, 0)
}
