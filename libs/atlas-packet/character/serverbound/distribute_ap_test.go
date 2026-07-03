package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/DistributeAp version=gms_v84 ida=0xa6f054
// packet-audit:verify packet=character/serverbound/DistributeAp version=gms_v87 ida=0xabb60b
// packet-audit:verify packet=character/serverbound/DistributeAp version=gms_v95 ida=0x9f61c0
// packet-audit:verify packet=character/serverbound/DistributeAp version=gms_v83 ida=0xa23b3d
// packet-audit:verify packet=character/serverbound/DistributeAp version=jms_v185 ida=0xb0ad8c
func TestDistributeApRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := DistributeAp{updateTime: 12345, dwFlag: 64}
			output := DistributeAp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.DwFlag() != input.DwFlag() {
				t.Errorf("dwFlag: got %v, want %v", output.DwFlag(), input.DwFlag())
			}
		})
	}
}

// TestDistributeApV79ByteOutput pins the gms_v79 DISTRIBUTE_AP (op 0x55) wire.
//
// Sender sub_96DB81 (GMS_v79_1_DEVM.exe @0x96db81):
//
//	COutPacket::COutPacket(v23, 85)  @0x96dca1 → opcode 85 (matches registry)
//	COutPacket::Encode4(v23, v10)    @0x96dcb3 → update_time (get_update_time @0x96db9e)
//	COutPacket::Encode4(v23, a2)     @0x96dcbe → dwFlag (the ability-up bitmask)
//
// Body = updateTime(4) + dwFlag(4) = 8 bytes. Version-invariant vs v83.
//
// packet-audit:verify packet=character/serverbound/DistributeAp version=gms_v48 ida=0x71cd00
// TestDistributeApV48ByteOutput pins the gms_v48 DISTRIBUTE_AP (op 67). IDA:
// CWvsContext::SendAbilityUpRequest = sub_71CD00 @0x71cd00 (GMS_v48_1_DEVM.exe)
// builds COutPacket(67) then Encode4(updateTime)@0x71cdb2 + Encode4(dwFlag)@0x71cdba
// — the exclusive-request tick IS present at v48 (unlike HEAL_OVER_TIME). Same shape
// as v79. No codec gate.
func TestDistributeApV48ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := DistributeAp{updateTime: 100, dwFlag: 0x20}
	expected := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)
		0x20, 0x00, 0x00, 0x00, // dwFlag 0x20 (Encode4)
	}
	if actual := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(actual, expected) {
		t.Errorf("v48 distribute-ap golden mismatch:\n got %x\nwant %x", actual, expected)
	}
}

// packet-audit:verify packet=character/serverbound/DistributeAp version=gms_v79 ida=0x96db81
func TestDistributeApV79ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := DistributeAp{updateTime: 100, dwFlag: 0x20}
	expected := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)  /*0x96dcb3*/
		0x20, 0x00, 0x00, 0x00, // dwFlag 0x20 (Encode4)     /*0x96dcbe*/
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 distribute-ap golden mismatch:\n got %x\nwant %x", actual, expected)
	}
}
