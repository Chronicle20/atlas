package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/DropMeso version=gms_v83 ida=0xa23de5
// packet-audit:verify packet=character/serverbound/DropMeso version=gms_v87 ida=0xabb8b3
// packet-audit:verify packet=character/serverbound/DropMeso version=gms_v95 ida=0x9f6650
// packet-audit:verify packet=character/serverbound/DropMeso version=gms_v84 ida=0xa6f482
// packet-audit:verify packet=character/serverbound/DropMeso version=jms_v185 ida=0xb0b14e
// packet-audit:verify packet=character/serverbound/DropMeso version=gms_v79 ida=0x96dfaf
func TestDropMesoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := DropMeso{updateTime: 100, amount: 5000}
			output := DropMeso{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Amount() != input.Amount() {
				t.Errorf("amount: got %v, want %v", output.Amount(), input.Amount())
			}
		})
	}
}

// TestDropMesoByteFixtureV79 pins the MESO_DROP (send op 92) wire against
// CWvsContext::SendDropMoneyRequest (v79 @0x96dfaf, byte-signature twin of v83
// @0xa23de5). After the drop-disabled field-flag guard the client emits
// COutPacket(92) + Encode4(update_time) + Encode4(nAmount):
//
//	updateTime = Encode4  /*0x96e033*/
//	amount     = Encode4  /*0x96e03e*/
func TestDropMesoByteFixtureV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	// updateTime=100 (0x00000064 LE), amount=5000 (0x00001388 LE)
	got := pt.Encode(t, ctx, DropMeso{updateTime: 100, amount: 5000}.Encode, nil)
	want := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime (Encode4) /*0x96e033*/
		0x88, 0x13, 0x00, 0x00, // amount (Encode4)     /*0x96e03e*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 bytes:\n got %x\nwant %x", got, want)
	}
}
