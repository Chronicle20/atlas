package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v72 ida=0x90cd28
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v79 ida=0x95e0f0
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v83 ida=0xa1288f
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v84 ida=0xa5cccc
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v87 ida=0xaa82b6
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v92 ida=0x9b0d20
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v95 ida=0x9db430
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=jms_v185 ida=0xaf8a40
func TestStoredExperienceUseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := StoredExperienceUse{updateTime: 0x0A0B0C0D}
			output := StoredExperienceUse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}

// TestStoredExperienceUseByteFixtureV95 pins the gms_v95 USE_GACHA_EXP wire.
//
// CWvsContext::SendTempExpUseRequest @0x9db430 builds COutPacket(182) then a
// single Encode4(get_update_time()) and nothing else, then SendPacket +
// SetExclRequestSent(1). The body is invariant across all eight columns.
func TestStoredExperienceUseByteFixtureV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	got := pt.Encode(t, ctx, StoredExperienceUse{updateTime: 0x0A0B0C0D}.Encode, nil)
	want := []byte{0x0D, 0x0C, 0x0B, 0x0A} // updateTime Encode4 (LE) /*0x9db430*/
	if !bytes.Equal(got, want) {
		t.Errorf("v95 bytes:\n got %x\nwant %x", got, want)
	}
}

// TestStoredExperienceUseByteFixtureV72 pins the gms_v72 USE_GACHA_EXP wire.
//
// CWvsContext::SendTempExpUseRequest @0x90cd28 has the identical shape: a
// single Encode4(updateTime) and nothing else. The op's defining property is
// that it carries nothing but the tick.
func TestStoredExperienceUseByteFixtureV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := pt.Encode(t, ctx, StoredExperienceUse{updateTime: 0x0A0B0C0D}.Encode, nil)
	want := []byte{0x0D, 0x0C, 0x0B, 0x0A} // updateTime Encode4 (LE) /*0x90cd28*/
	if !bytes.Equal(got, want) {
		t.Errorf("v72 bytes:\n got %x\nwant %x", got, want)
	}
	if len(got) != 4 {
		t.Fatalf("v72 length: got %d, want 4", len(got))
	}
}
