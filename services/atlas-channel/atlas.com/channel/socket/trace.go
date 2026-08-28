package socket

import (
	"atlas-channel/configuration"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"
	socket "github.com/Chronicle20/atlas/libs/atlas-socket"
	"github.com/Chronicle20/atlas/libs/atlas-socket/trace"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// NewPacketTracer builds the inbound tracer this tenant's listener installs
// on libs/atlas-socket. The closure captures t, so per-tenant scoping is
// structural: each listener gets its own tracer bound to its own tenant and
// there is no path by which tenant B's flag reaches tenant A's socket
// (FR-2.6).
//
// names is the op -> configured handler name map for this service's slice
// of the tenant socket config. An opcode with no entry renders "<none>"
// rather than being dropped, because an unhandled opcode is exactly the
// case the trace exists to diagnose (FR-3.4).
func NewPacketTracer(l logrus.FieldLogger, t tenant.Model, names map[uint16]string) socket.PacketTracer {
	opSize := opcodes.OpCodeSize(t.Region(), t.MajorVersion())
	return func(sessionId uuid.UUID, op uint16, payload []byte) {
		// A trace failure must never fail the packet it describes (FR-2.5).
		defer func() {
			if r := recover(); r != nil {
				l.Warnf("Packet trace panicked and was suppressed: %v", r)
			}
		}()
		if !trace.Enabled(l, configuration.TracePacketsEnabled(t.Id())) {
			return
		}
		name, ok := names[op]
		if !ok {
			name = "<none>"
		}
		l.Debug(trace.Format(trace.Header{
			Direction: trace.Inbound,
			Name:      name,
			Op:        &op,
			OpSize:    opSize,
			Length:    len(payload),
			SessionId: sessionId,
		}, payload))
	}
}
