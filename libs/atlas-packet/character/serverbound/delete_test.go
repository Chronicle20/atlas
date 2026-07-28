package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/DeleteCharacter version=gms_v48 ida=0x50043f
// packet-audit:verify packet=character/serverbound/DeleteCharacter version=gms_v83 ida=0x5f7c4a
// packet-audit:verify packet=character/serverbound/DeleteCharacter version=gms_v87 ida=0x62f3d3
// packet-audit:verify packet=character/serverbound/DeleteCharacter version=gms_v95 ida=0x5d53a0
// packet-audit:verify packet=character/serverbound/DeleteCharacter version=gms_v84 ida=0x60cbc0
// packet-audit:verify packet=character/serverbound/DeleteCharacter version=jms_v185 ida=0x66e0f9
func TestDeleteCharacterRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := DeleteCharacter{
				pic:         "123456",
				dob:         19900101,
				characterId: 12345,
			}
			output := DeleteCharacter{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if v.Region == "GMS" && v.MajorVersion > 82 {
				if output.Pic() != input.Pic() {
					t.Errorf("pic: got %v, want %v", output.Pic(), input.Pic())
				}
			} else if v.Region == "GMS" {
				if output.Dob() != input.Dob() {
					t.Errorf("dob: got %v, want %v", output.Dob(), input.Dob())
				}
			}
		})
	}
}

// TestDeleteCharacterJMSGolden pins the exact jms_v185 wire for DeleteCharacter
// against CLogin::SendDeleteCharPacket @0x66e0f9: COutPacket(0xD) then
// Encode4(selected avatar's char id) — a single 4-byte int, NO PIC and NO DOB
// (the GMS-only prefixes do not fire for JMS).
// TestDeleteCharacterV48ByteOutput pins the gms_v48 DELETE_CHAR (op 22). IDA:
// CLogin::SendDeleteCharPacket = sub_50043F @0x50043f (GMS_v48_1_DEVM.exe) builds
// COutPacket(22) then Encode4(dob)@0x5004bb + Encode4(charId)@0x5004c9 — the legacy
// GMS (<83) DOB-then-charId shape (no PIC string). Matches the codec's <=82 branch.
func TestDeleteCharacterV48ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	got := DeleteCharacter{dob: 100, characterId: 12345}.Encode(nil, ctx)(nil)
	want := []byte{
		0x64, 0x00, 0x00, 0x00, // Encode4 dob = 100      /*0x5004bb*/
		0x39, 0x30, 0x00, 0x00, // Encode4 charId = 12345 /*0x5004c9*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 DeleteCharacter wire: got %x want %x", got, want)
	}
}

func TestDeleteCharacterJMSGolden(t *testing.T) {
	v := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	got := DeleteCharacter{pic: "123456", dob: 19900101, characterId: 12345}.Encode(nil, ctx)(nil)
	want := []byte{0x39, 0x30, 0x00, 0x00} // Encode4(12345) little-endian
	if !bytes.Equal(got, want) {
		t.Errorf("jms DeleteCharacter wire: got %x want %x", got, want)
	}
}
