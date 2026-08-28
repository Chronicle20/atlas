// Package trace formats raw packet bytes into a human-readable, hex+ASCII
// trace line for diagnostic logging. It is intentionally self-contained: it
// imports nothing from the sibling socket package so that it can be reused
// by any packet-handling code without pulling in session/connection state.
package trace

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Direction identifies whether a packet was received from, or sent to, the
// remote peer.
type Direction int

const (
	// Inbound marks a packet read from the socket (already decrypted).
	Inbound Direction = iota
	// Outbound marks a packet about to be written to the socket (not yet
	// encrypted or header-prefixed).
	Outbound
)

// Header carries the metadata rendered on the trace line above the hex dump.
type Header struct {
	Direction Direction
	Name      string
	// Op is a pointer because not every packet has an opcode: the
	// WriteHello handshake response has none (FR-4.3).
	Op        *uint16
	OpSize    int
	Length    int
	SessionId uuid.UUID
}

// Format renders a Header and its packet body as a single string: the
// header line, then (unless the body is empty) a newline followed by the
// hex+ASCII dump of the body. It is returned as one string, not logged as
// separate lines, because concurrent sessions on the same pod would
// otherwise interleave their bodies in the log stream (FR-5.7).
func Format(h Header, b []byte) string {
	header := formatHeader(h)
	if len(b) == 0 {
		return header
	}
	return header + "\n" + Dump(b)
}

func formatHeader(h Header) string {
	var dirTag, nameTag string
	if h.Direction == Inbound {
		dirTag = "[PKT IN ]"
		nameTag = "handler=" + h.Name
	} else {
		dirTag = "[PKT OUT]"
		nameTag = "writer=" + h.Name
	}

	var opTag string
	if h.Op == nil {
		opTag = "op=n/a"
	} else if h.OpSize == 1 {
		opTag = fmt.Sprintf("op=0x%02x", *h.Op)
	} else {
		opTag = fmt.Sprintf("op=0x%04x", *h.Op)
	}

	return fmt.Sprintf("%s %s %s len=%d session=%s", dirTag, nameTag, opTag, h.Length, h.SessionId.String())
}

// Dump renders b as a hex+ASCII dump, one 16-byte chunk per line, joined by
// "\n" with no trailing newline. It never truncates (FR-5.5): every packet,
// however large, is rendered in full.
func Dump(bs []byte) string {
	if len(bs) == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow((len(bs)/16 + 1) * 78)

	for o := 0; o < len(bs); o += 16 {
		end := o + 16
		if end > len(bs) {
			end = len(bs)
		}
		chunk := bs[o:end]

		if o > 0 {
			b.WriteByte('\n')
		}

		fmt.Fprintf(&b, "%04x  ", o)
		for i := 0; i < 16; i++ {
			if i == 8 {
				b.WriteByte(' ')
			}
			if i < len(chunk) {
				fmt.Fprintf(&b, "%02x ", chunk[i])
			} else {
				b.WriteString("   ")
			}
		}
		b.WriteByte(' ')
		b.WriteByte('|')
		for _, c := range chunk {
			if c >= 0x20 && c <= 0x7e {
				b.WriteByte(c)
			} else {
				b.WriteByte('.')
			}
		}
		b.WriteByte('|')
	}

	return b.String()
}

// Enabled reports whether packet tracing should fire for this call. It
// checks flag first and returns immediately when it is false, before ever
// touching the logger, because the disabled path is the hot path and must
// not allocate or dispatch (FR-2.1). When flag is true, it then asks the
// logger whether Debug is enabled so a trace is never built only to be
// discarded by the log level. If the logger does not expose a level probe,
// it errs toward tracing (best effort): logrus itself will discard the
// entry if the level does not warrant it.
func Enabled(l logrus.FieldLogger, flag bool) bool {
	if !flag {
		return false
	}
	if lc, ok := l.(interface {
		IsLevelEnabled(logrus.Level) bool
	}); ok {
		return lc.IsLevelEnabled(logrus.DebugLevel)
	}
	return true
}
