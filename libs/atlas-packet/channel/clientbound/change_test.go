package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestChannelChangeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ChannelChange{ipAddr: "192.168.1.1", port: 7575}
			output := ChannelChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.IpAddr() != input.IpAddr() {
				t.Errorf("ipAddr: got %v, want %v", output.IpAddr(), input.IpAddr())
			}
			if output.Port() != input.Port() {
				t.Errorf("port: got %v, want %v", output.Port(), input.Port())
			}
		})
	}
}

// TestChannelChangeWireShape proves the exact wire layout matches what
// CClientSocket::OnMigrateCommand (GMS v95 @ 0x4add50) reads:
//
//	Decode1 (success flag) + Decode4 (IP as raw uint32) + Decode2 (port) = 7 bytes.
//
// All versions share the same layout — no version gate needed.
func TestChannelChangeWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := ChannelChange{ipAddr: "10.0.0.1", port: 8484}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// 1 (flag) + 4 (IP octets) + 2 (port) = 7 bytes
			if len(b) != 7 {
				t.Errorf("wire size = %d bytes, want 7: % x", len(b), b)
			}
			// Byte 0 must be the literal success flag (1)
			if b[0] != 1 {
				t.Errorf("byte[0] = 0x%02x, want 0x01 (success flag)", b[0])
			}
			// Bytes 1-4 must be IP octets in network order: 10, 0, 0, 1
			if b[1] != 10 || b[2] != 0 || b[3] != 0 || b[4] != 1 {
				t.Errorf("IP bytes = % x, want [0a 00 00 01]", b[1:5])
			}
		})
	}
}
