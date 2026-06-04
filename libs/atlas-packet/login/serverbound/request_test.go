package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/sirupsen/logrus"
)

func TestRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Request{
				name:           "testuser",
				password:       "testpass",
				hwid:           make([]byte, 16),
				gameRoomClient: 42,
				gameStartMode:  1,
				unknown1:       2,
				unknown2:       3,
				partnerCode:    0xDEADBEEF,
			}
			output := Request{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Password() != input.Password() {
				t.Errorf("password: got %v, want %v", output.Password(), input.Password())
			}
			if output.GameRoomClient() != input.GameRoomClient() {
				t.Errorf("gameRoomClient: got %v, want %v", output.GameRoomClient(), input.GameRoomClient())
			}
			if output.GameStartMode() != input.GameStartMode() {
				t.Errorf("gameStartMode: got %v, want %v", output.GameStartMode(), input.GameStartMode())
			}
			// PartnerCode round-trips only for GMS (JMS omits it from the wire).
			wantPartner := input.PartnerCode()
			if v.Region != "GMS" {
				wantPartner = 0
			}
			if output.PartnerCode() != wantPartner {
				t.Errorf("partnerCode: got %v, want %v", output.PartnerCode(), wantPartner)
			}
		})
	}
}

// TestRequestTrailerShape asserts the exact on-wire trailing layout per version
// (IDA harvest, task-080 B6.1):
//   - GMS v83/v87/v95: ...gameStartMode, unknown1, unknown2, Encode4(PartnerCode)  (2 bytes + 4)
//   - JMS185:          ...gameStartMode, unknown1                                   (1 byte, no PartnerCode)
func TestRequestTrailerShape(t *testing.T) {
	l, _ := logrusNull()

	const name = "testuser"
	const password = "testpass"
	// Fixed-length prefix consumed before the trailer:
	//   WriteAsciiString(name)     = 2 + len(name)
	//   WriteAsciiString(password) = 2 + len(password)
	//   WriteByteArray(hwid[16])   = 16
	//   WriteInt(gameRoomClient)   = 4
	//   WriteByte(gameStartMode)   = 1
	prefixLen := (2 + len(name)) + (2 + len(password)) + 16 + 4 + 1

	cases := []struct {
		name        string
		region      string
		major       uint16
		wantTrailer []byte // bytes after the fixed prefix
	}{
		{"GMS v83", "GMS", 83, []byte{0x07, 0x09, 0xEF, 0xBE, 0xAD, 0xDE}},
		{"GMS v87", "GMS", 87, []byte{0x07, 0x09, 0xEF, 0xBE, 0xAD, 0xDE}},
		{"GMS v95", "GMS", 95, []byte{0x07, 0x09, 0xEF, 0xBE, 0xAD, 0xDE}},
		{"JMS v185", "JMS", 185, []byte{0x07}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := pt.CreateContext(c.region, c.major, 1)
			in := Request{
				name:           name,
				password:       password,
				hwid:           make([]byte, 16),
				gameRoomClient: 1,
				gameStartMode:  0,
				unknown1:       0x07,
				unknown2:       0x09,
				partnerCode:    0xDEADBEEF,
			}
			b := in.Encode(l, ctx)(nil)
			if len(b) != prefixLen+len(c.wantTrailer) {
				t.Fatalf("total length: got %d, want %d (prefix %d + trailer %d)",
					len(b), prefixLen+len(c.wantTrailer), prefixLen, len(c.wantTrailer))
			}
			trailer := b[prefixLen:]
			for i, want := range c.wantTrailer {
				if trailer[i] != want {
					t.Errorf("trailer[%d]: got 0x%02X, want 0x%02X (full trailer %X)",
						i, trailer[i], want, trailer)
				}
			}
		})
	}
}

func logrusNull() (logrus.FieldLogger, func()) {
	l := logrus.New()
	l.Out = nil
	l.SetOutput(devNull{})
	return l, func() {}
}

type devNull struct{}

func (devNull) Write(p []byte) (int, error) { return len(p), nil }
