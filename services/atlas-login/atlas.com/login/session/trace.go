package session

import (
	"atlas-login/configuration"
	"encoding/binary"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"
	"github.com/Chronicle20/atlas/libs/atlas-socket/trace"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// tracePacketOut logs the plaintext bytes a writer produced for name,
// before they reach announceEncrypted (FR-4.2, FR-4.4). b is the writer's
// raw output: writer.MessageGetter puts the opcode, when the packet has
// one, in the first 1-2 bytes, little-endian for the 2-byte form, so the
// opcode is read straight off the payload rather than threaded through
// from the caller.
func tracePacketOut(l logrus.FieldLogger, t tenant.Model, name string, sessionId uuid.UUID, b []byte) {
	// A trace failure must never fail the packet it describes (FR-2.5).
	defer func() {
		if r := recover(); r != nil {
			l.Warnf("Packet trace panicked and was suppressed: %v", r)
		}
	}()
	if !trace.Enabled(l, configuration.TracePacketsEnabled(t.Id())) {
		return
	}

	opSize := opcodes.OpCodeSize(t.Region(), t.MajorVersion())

	var op *uint16
	if name == "<hello>" {
		// The handshake frame is not an opcode-tagged packet; it has no
		// opcode to read off the payload.
		op = nil
	} else if opSize == 1 && len(b) >= 1 {
		v := uint16(b[0])
		op = &v
	} else if opSize == 2 && len(b) >= 2 {
		v := binary.LittleEndian.Uint16(b[0:2])
		op = &v
	}

	l.Debug(trace.Format(trace.Header{
		Direction: trace.Outbound,
		Name:      name,
		Op:        op,
		OpSize:    opSize,
		Length:    len(b),
		SessionId: sessionId,
	}, b))
}
