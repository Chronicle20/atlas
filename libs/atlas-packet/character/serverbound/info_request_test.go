package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/InfoRequest version=gms_v83 ida=0xa23fba
// packet-audit:verify packet=character/serverbound/InfoRequest version=gms_v87 ida=0xabba88
// packet-audit:verify packet=character/serverbound/InfoRequest version=gms_v95 ida=0x9f2f70
// packet-audit:verify packet=character/serverbound/InfoRequest version=jms_v185 ida=0xb0b323
// packet-audit:verify packet=character/serverbound/InfoRequest version=gms_v84 ida=0xa6f657
func TestInfoRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := InfoRequest{updateTime: 100, characterId: 12345, petInfo: true}
			output := InfoRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.PetInfo() != input.PetInfo() {
				t.Errorf("petInfo: got %v, want %v", output.PetInfo(), input.PetInfo())
			}
		})
	}
}

// TestInfoRequestV79ByteOutput pins the gms_v79 CHAR_INFO_REQUEST (op 0x5F) wire.
//
// Sender sub_96E184 (GMS_v79_1_DEVM.exe @0x96e184):
//
//	COutPacket::COutPacket(v11, 95)  @0x96e1e1 → opcode 95 (matches registry)
//	COutPacket::Encode4(v11, v8)     @0x96e1f8 → update_time (get_update_time @0x96e19c)
//	COutPacket::Encode4(v11, v6)     @0x96e201 → characterId (a2)
//	COutPacket::Encode1(v11, a4)     @0x96e20c → petInfo bool
//
// Body = updateTime(4) + characterId(4) + petInfo(1) = 9 bytes. Version-invariant vs v83.
//
// packet-audit:verify packet=character/serverbound/InfoRequest version=gms_v79 ida=0x96e184
func TestInfoRequestV79ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := InfoRequest{updateTime: 100, characterId: 12345, petInfo: true}
	expected := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)        /*0x96e1f8*/
		0x39, 0x30, 0x00, 0x00, // characterId 12345=0x3039 (Enc4) /*0x96e201*/
		0x01, // petInfo true (Encode1)                            /*0x96e20c*/
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 info-request golden mismatch:\n got %x\nwant %x", actual, expected)
	}
}
