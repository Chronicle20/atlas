package clientbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestHelloRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewHello(83, 1, []byte{1, 2, 3, 4}, []byte{5, 6, 7, 8}, 8)
			output := Hello{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.MajorVersion() != input.MajorVersion() {
				t.Errorf("majorVersion: got %v, want %v", output.MajorVersion(), input.MajorVersion())
			}
			if output.MinorVersion() != input.MinorVersion() {
				t.Errorf("minorVersion: got %v, want %v", output.MinorVersion(), input.MinorVersion())
			}
			if output.Locale() != input.Locale() {
				t.Errorf("locale: got %v, want %v", output.Locale(), input.Locale())
			}
			for i := range input.SendIv() {
				if output.SendIv()[i] != input.SendIv()[i] {
					t.Errorf("sendIv[%d]: got %v, want %v", i, output.SendIv()[i], input.SendIv()[i])
				}
			}
			for i := range input.RecvIv() {
				if output.RecvIv()[i] != input.RecvIv()[i] {
					t.Errorf("recvIv[%d]: got %v, want %v", i, output.RecvIv()[i], input.RecvIv()[i])
				}
			}
		})
	}
}

// TestHelloWireShape proves the exact wire layout matches what
// CClientSocket::OnConnect (GMS v95 @ 0x4aef10) decodes:
//
//	Decode2 (packet length, 0x000E)
//	Decode2 (majorVersion uint16 LE)
//	DecodeStr (minorVersion as ASCII string, 2-byte length prefix)
//	Decode4 (recvIv → client m_uSeqSnd, 4 bytes)
//	Decode4 (sendIv → client m_uSeqRcv, 4 bytes)
//	Decode1 (locale byte)
//
// recvIv precedes sendIv on the wire — consistent across all versions.
func TestHelloWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	recvIv := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	sendIv := []byte{0x11, 0x22, 0x33, 0x44}
	in := NewHello(95, 1, sendIv, recvIv, 8)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// Expected layout:
			//   [0..1]  length 0x000E (2 bytes, LE)
			//   [2..3]  majorVersion = 95 (2 bytes, LE)
			//   [4..5]  string length = 1 (2 bytes, LE, length prefix of "1")
			//   [6]     '1' (ASCII)
			//   [7..10] recvIv (4 bytes)
			//   [11..14] sendIv (4 bytes)
			//   [15]    locale = 8 (1 byte)
			// Total: 16 bytes
			if len(b) != 16 {
				t.Fatalf("wire size = %d bytes, want 16: % x", len(b), b)
			}
			// offset 0: length header
			if gotLen := binary.LittleEndian.Uint16(b[0:2]); gotLen != 0x000E {
				t.Errorf("length header = 0x%04x, want 0x000E", gotLen)
			}
			// offset 2: majorVersion
			if gotMajor := binary.LittleEndian.Uint16(b[2:4]); gotMajor != 95 {
				t.Errorf("majorVersion = %d, want 95", gotMajor)
			}
			// offset 4: minorVersion string ("1" → 2-byte len + 1 byte data)
			if gotStrLen := binary.LittleEndian.Uint16(b[4:6]); gotStrLen != 1 {
				t.Errorf("minorVersion string length = %d, want 1", gotStrLen)
			}
			if b[6] != '1' {
				t.Errorf("minorVersion char = %q, want '1'", b[6])
			}
			// offset 7: recvIv (first IV on wire → client m_uSeqSnd)
			for i, want := range recvIv {
				if b[7+i] != want {
					t.Errorf("recvIv[%d] = 0x%02x, want 0x%02x", i, b[7+i], want)
				}
			}
			// offset 11: sendIv (second IV on wire → client m_uSeqRcv)
			for i, want := range sendIv {
				if b[11+i] != want {
					t.Errorf("sendIv[%d] = 0x%02x, want 0x%02x", i, b[11+i], want)
				}
			}
			// offset 15: locale
			if b[15] != 8 {
				t.Errorf("locale = %d, want 8", b[15])
			}
		})
	}
}
