package serverbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestChannelConnectRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ChannelConnect{
				characterId: 12345,
				machineId:   make([]byte, 16),
				gm:          true,
				unknown1:    false,
				unknown2:    99999,
			}
			output := ChannelConnect{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.Gm() != input.Gm() {
				t.Errorf("gm: got %v, want %v", output.Gm(), input.Gm())
			}
			if output.Unknown1() != input.Unknown1() {
				t.Errorf("unknown1: got %v, want %v", output.Unknown1(), input.Unknown1())
			}
			if output.Unknown2() != input.Unknown2() {
				t.Errorf("unknown2: got %v, want %v", output.Unknown2(), input.Unknown2())
			}
		})
	}
}

// TestChannelConnectWireShape proves the JMS vs GMS wire layout difference for the
// gm/dummy1 field.
//
// JMS v185 CClientSocket::OnConnect (non-login branch @ 0x4b051f):
//   - COutPacket::Encode4(characterId)       → 4 bytes
//   - COutPacket::EncodeBuffer(MachineId,16) → 16 bytes
//   - COutPacket::Encode2(dummy1)            → 2 bytes  ← JMS gm field is uint16
//   - COutPacket::Encode1(0)                 → 1 byte
//   - COutPacket::EncodeBuffer(unknown2, 8)  → 8 bytes
//   Total: 4+16+2+1+8 = 31 bytes
//
// GMS (all versions) uses Encode1 for gm, so:
//   Total: 4+16+1+1+8 = 30 bytes
func TestChannelConnectWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ChannelConnect{
		characterId: 1,
		machineId:   make([]byte, 16),
		gm:          true,
		unknown1:    false,
		unknown2:    0,
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)

			if v.Region == "JMS" {
				// 4 + 16 + 2 + 1 + 8 = 31 bytes
				if len(b) != 31 {
					t.Errorf("JMS wire size = %d bytes, want 31: % x", len(b), b)
				}
				// gm field at offset 20 is a little-endian uint16
				gotGm := binary.LittleEndian.Uint16(b[20:22])
				if gotGm != 1 {
					t.Errorf("JMS gm uint16 = %d, want 1", gotGm)
				}
			} else {
				// 4 + 16 + 1 + 1 + 8 = 30 bytes
				if len(b) != 30 {
					t.Errorf("GMS wire size = %d bytes, want 30: % x", len(b), b)
				}
				// gm field at offset 20 is a single byte
				if b[20] != 0x01 {
					t.Errorf("GMS gm byte = 0x%02x, want 0x01", b[20])
				}
			}
		})
	}
}
