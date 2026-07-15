package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/ChalkboardUse version=gms_v83 ida=0x937607
// packet-audit:verify packet=character/clientbound/ChalkboardUse version=gms_v87 ida=0x9b1d1e
// packet-audit:verify packet=character/clientbound/ChalkboardUse version=gms_v95 ida=0x8ed310
// packet-audit:verify packet=character/clientbound/ChalkboardUse version=jms_v185 ida=0x9f6199
// packet-audit:verify packet=character/clientbound/ChalkboardUse version=gms_v84 ida=0x96e8c0
// packet-audit:verify packet=character/clientbound/ChalkboardUse version=gms_v79 ida=0x890f5a
func TestChalkboardUse(t *testing.T) {
	input := NewChalkboardUse(1234, "Selling scrolls!")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestChalkboardClear(t *testing.T) {
	input := NewChalkboardClear(1234)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestChalkboardUseByteFixtureV79 pins the exact CHALKBOARD (op 153) wire bytes
// against CUser::OnADBoard (v79 @0x890f5a). The leading characterId int is read
// by CUserPool::OnUserCommonPacket before dispatch into OnADBoard; the codec
// prepends it. OnADBoard read order:
//
//	characterId = Decode4  // consumed by dispatcher before OnADBoard
//	active      = Decode1  // if-guard @0x890f89 (0 => clear, no message)
//	message     = DecodeStr // ZXString length-prefixed, only when active /*0x890fbc*/
func TestChalkboardUseByteFixtureV79(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)

	t.Run("active", func(t *testing.T) {
		// characterId=1234 (0x000004D2 LE), active=1, message="Hi" (len 2)
		got := NewChalkboardUse(1234, "Hi").Encode(nil, ctx)(nil)
		want := []byte{
			0xD2, 0x04, 0x00, 0x00, // characterId (dispatcher prefix)
			0x01,       // active = 1 (Decode1) /*0x890f89*/
			0x02, 0x00, // message len = 2 (DecodeStr short) /*0x890fbc*/
			0x48, 0x69, // "Hi"                              /*0x890fbc*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("active bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("clear", func(t *testing.T) {
		// characterId=1234, active=0 (no message read on the clear path)
		got := NewChalkboardClear(1234).Encode(nil, ctx)(nil)
		want := []byte{
			0xD2, 0x04, 0x00, 0x00, // characterId (dispatcher prefix)
			0x00, // active = 0 (Decode1 false) /*0x890f89 -> else @0x890f96*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("clear bytes:\n got %x\nwant %x", got, want)
		}
	})
}
