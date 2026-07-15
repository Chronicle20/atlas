package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/DistributeSp version=gms_v83 ida=0xa23cf3
// packet-audit:verify packet=character/serverbound/DistributeSp version=gms_v87 ida=0xabb7c1
// packet-audit:verify packet=character/serverbound/DistributeSp version=gms_v95 ida=0x9f2e90
// packet-audit:verify packet=character/serverbound/DistributeSp version=gms_v84 ida=0xa6f390
// packet-audit:verify packet=character/serverbound/DistributeSp version=jms_v185 ida=0xb0b0c8
func TestDistributeSpRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := DistributeSp{updateTime: 12345, skillId: 1001004}
			output := DistributeSp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.SkillId() != input.SkillId() {
				t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
			}
		})
	}
}

// TestDistributeSpV79ByteOutput pins the gms_v79 DISTRIBUTE_SP (op 0x58) wire.
//
// Sender sub_96DEBD (GMS_v79_1_DEVM.exe @0x96debd):
//
//	COutPacket::COutPacket(v8, 88)  @0x96dee2 → opcode 88 (matches registry)
//	COutPacket::Encode4(v8, v5)     @0x96def4 → update_time (get_update_time @0x96ded4)
//	COutPacket::Encode4(v8, a2)     @0x96deff → skillId
//
// Body = updateTime(4) + skillId(4) = 8 bytes. Version-invariant vs v83.
//
// packet-audit:verify packet=character/serverbound/DistributeSp version=gms_v48 ida=0x71ceb3
// TestDistributeSpV48ByteOutput pins the gms_v48 DISTRIBUTE_SP (op 73). IDA:
// CWvsContext::SendSkillUpRequest = sub_71CEB3 @0x71ceb3 (GMS_v48_1_DEVM.exe) builds
// COutPacket(73) then Encode4(updateTime)@0x71cee6 + Encode4(skillId)@0x71ceee. Same
// shape as v79. No codec gate.
func TestDistributeSpV48ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := DistributeSp{updateTime: 100, skillId: 1000000}
	expected := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)
		0x40, 0x42, 0x0F, 0x00, // skillId 1000000=0xF4240 (Encode4)
	}
	if actual := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(actual, expected) {
		t.Errorf("v48 distribute-sp golden mismatch:\n got %x\nwant %x", actual, expected)
	}
}

// packet-audit:verify packet=character/serverbound/DistributeSp version=gms_v79 ida=0x96debd
func TestDistributeSpV79ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := DistributeSp{updateTime: 100, skillId: 1000000}
	expected := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)          /*0x96def4*/
		0x40, 0x42, 0x0F, 0x00, // skillId 1000000=0xF4240 (Encode4) /*0x96deff*/
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 distribute-sp golden mismatch:\n got %x\nwant %x", actual, expected)
	}
}
