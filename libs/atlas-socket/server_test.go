package socket

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// header builds a 4-byte frame header that PacketLength decodes to bodyLen.
// PacketLength is (b0 + b1*0x100) ^ (b2 + b3*0x100), so leaving the first pair
// zero makes the second pair the length outright.
func header(bodyLen int) []byte {
	return []byte{0x00, 0x00, byte(bodyLen & 0xFF), byte(bodyLen >> 8 & 0xFF)}
}

// frame is a header plus a body of opcode (little-endian uint16) + payload.
func frame(op uint16, payload []byte) []byte {
	body := append([]byte{byte(op & 0xFF), byte(op >> 8 & 0xFF)}, payload...)
	return append(header(len(body)), body...)
}

type received struct {
	op      uint16
	payload []byte
}

// serve wires run() to one end of a net.Pipe and reports every handled packet.
// net.Pipe is unbuffered and returns exactly what each Write supplies, which is
// what lets a test split a frame across writes the way TCP splits it across
// segments.
func serve(t *testing.T, ops []uint16) (net.Conn, <-chan received, func()) {
	t.Helper()

	client, server := net.Pipe()
	got := make(chan received, 8)

	handlers := make(map[uint16]request.Handler)
	for _, op := range ops {
		op := op
		handlers[op] = func(_ uuid.UUID, r request.Reader) {
			got <- received{op: op, payload: r.GetRestAsBytes()}
		}
	}

	l := logrus.New()
	l.SetOutput(nopWriter{})

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	cfg := &config{
		rw:        ShortReadWriter{},
		creator:   defaultCreator,
		decryptor: defaultMessageDecryptor,
		destroyer: defaultDestroyer,
		handlers:  handlers,
	}

	routine.Go(l, ctx, func(_ context.Context) { run(l, ctx, wg)(cfg, server, uuid.New(), 4) })

	return client, got, func() {
		cancel()
		_ = client.Close()
	}
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func await(t *testing.T, got <-chan received) received {
	t.Helper()
	select {
	case r := <-got:
		return r
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for a handled packet")
		return received{}
	}
}

// TestReadAssemblesFrameSplitAcrossReads is the regression test for the short-read
// bug: a body larger than one TCP segment arrives across several reads, and taking
// the first read as the whole frame both truncated the payload (the untransferred
// tail stayed zero, which AES-OFB turns into raw keystream) and left the remainder
// in the socket, where it was consumed as the next frame's header — desynchronising
// the stream for the rest of the connection.
func TestReadAssemblesFrameSplitAcrossReads(t *testing.T) {
	client, got, stop := serve(t, []uint16{0x0018, 0x0001})
	defer stop()

	// A crash-report relay: 1640 payload bytes, so a 1644-byte body — larger than
	// a typical 1460-byte segment, which is what made this reachable in practice.
	payload := make([]byte, 1640)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	big := frame(0x0018, payload)

	// Header first, then the body in two writes, mimicking segmentation.
	split := 4 + 1456
	writeAll(t, client, big[:4])
	writeAll(t, client, big[4:split])
	writeAll(t, client, big[split:])

	first := await(t, got)
	if first.op != 0x0018 {
		t.Fatalf("opcode = 0x%04X, want 0x0018", first.op)
	}
	if len(first.payload) != len(payload) {
		t.Fatalf("payload length = %d, want %d", len(first.payload), len(payload))
	}
	for i := range payload {
		if first.payload[i] != payload[i] {
			t.Fatalf("payload byte %d = 0x%02X, want 0x%02X (truncated frame)", i, first.payload[i], payload[i])
		}
	}

	// The decisive half: if any of the oversized frame were left unconsumed, this
	// next frame's header would be read from those leftovers and the opcode would
	// be garbage.
	writeAll(t, client, frame(0x0001, []byte{0xAA, 0xBB}))

	second := await(t, got)
	if second.op != 0x0001 {
		t.Fatalf("second opcode = 0x%04X, want 0x0001 — stream desynchronised", second.op)
	}
	if len(second.payload) != 2 || second.payload[0] != 0xAA || second.payload[1] != 0xBB {
		t.Fatalf("second payload = % X, want AA BB", second.payload)
	}
}

// TestReadHandlesByteAtATimeDelivery is the pathological case: every read returns
// a single byte. Nothing about frame assembly may depend on how the bytes are
// grouped.
func TestReadHandlesByteAtATimeDelivery(t *testing.T) {
	client, got, stop := serve(t, []uint16{0x0002})
	defer stop()

	f := frame(0x0002, []byte{0x01, 0x02, 0x03, 0x04, 0x05})
	for i := range f {
		writeAll(t, client, f[i:i+1])
	}

	r := await(t, got)
	if r.op != 0x0002 {
		t.Fatalf("opcode = 0x%04X, want 0x0002", r.op)
	}
	if len(r.payload) != 5 {
		t.Fatalf("payload = % X, want 5 bytes", r.payload)
	}
}

func writeAll(t *testing.T, c net.Conn, b []byte) {
	t.Helper()
	_ = c.SetWriteDeadline(time.Now().Add(3 * time.Second))
	if _, err := c.Write(b); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// TestHandle_NilTracerIsSafe asserts the nil check is the entire cost when
// tracing is off (FR-2.1): a config with no tracer installed must not panic
// and must still dispatch to the registered handler.
func TestHandle_NilTracerIsSafe(t *testing.T) {
	l := logrus.New()
	l.SetOutput(nopWriter{})

	called := false
	cfg := &config{
		rw: ShortReadWriter{},
		handlers: map[uint16]request.Handler{
			0x0001: func(uuid.UUID, request.Reader) { called = true },
		},
	}

	handle(l)(cfg, uuid.New(), request.Request{0x01, 0x00, 0xaa})

	if !called {
		t.Fatal("handler did not run")
	}
}

// TestHandle_TracerSeesFullFrameBeforeHandler asserts the tracer receives
// the full decrypted frame -- including for an unregistered opcode
// (FR-3.4) -- and runs before the handler (FR-3.3).
func TestHandle_TracerSeesFullFrameBeforeHandler(t *testing.T) {
	tests := []struct {
		name        string
		registered  bool
		frame       request.Request
		wantOp      uint16
		wantPayload []byte
		wantHandled bool
	}{
		{
			name:        "registered handler",
			registered:  true,
			frame:       request.Request{0x01, 0x00, 0xaa, 0xbb},
			wantOp:      0x0001,
			wantPayload: []byte{0x01, 0x00, 0xaa, 0xbb},
			wantHandled: true,
		},
		{
			name:        "unregistered opcode",
			registered:  false,
			frame:       request.Request{0xff, 0x00, 0x11},
			wantOp:      0x00ff,
			wantPayload: []byte{0xff, 0x00, 0x11},
			wantHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := logrus.New()
			l.SetOutput(nopWriter{})

			var order []string
			handled := false

			handlers := map[uint16]request.Handler{}
			if tt.registered {
				handlers[0x0001] = func(uuid.UUID, request.Reader) {
					order = append(order, "handle")
					handled = true
				}
			}

			var tracedSessionId uuid.UUID
			var tracedOp uint16
			var tracedPayload []byte
			tracer := PacketTracer(func(sessionId uuid.UUID, op uint16, payload []byte) {
				order = append(order, "trace")
				tracedSessionId = sessionId
				tracedOp = op
				tracedPayload = payload
			})

			cfg := &config{
				rw:       ShortReadWriter{},
				handlers: handlers,
				tracer:   tracer,
			}

			sessionId := uuid.New()
			handle(l)(cfg, sessionId, tt.frame)

			if tracedOp != tt.wantOp {
				t.Fatalf("traced op = 0x%04X, want 0x%04X", tracedOp, tt.wantOp)
			}
			if string(tracedPayload) != string(tt.wantPayload) {
				t.Fatalf("traced payload = % X, want % X", tracedPayload, tt.wantPayload)
			}
			if tracedSessionId != sessionId {
				t.Fatalf("traced sessionId = %s, want %s", tracedSessionId, sessionId)
			}
			if handled != tt.wantHandled {
				t.Fatalf("handled = %v, want %v", handled, tt.wantHandled)
			}
			if tt.wantHandled {
				want := []string{"trace", "handle"}
				if len(order) != len(want) || order[0] != want[0] || order[1] != want[1] {
					t.Fatalf("order = %v, want %v", order, want)
				}
			}
		})
	}
}
