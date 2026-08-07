package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/HealOverTime version=gms_v83 ida=0xa1e997
// packet-audit:verify packet=character/serverbound/HealOverTime version=gms_v87 ida=0xab5ca8
// packet-audit:verify packet=character/serverbound/HealOverTime version=gms_v95 ida=0x9f2a00
// packet-audit:verify packet=character/serverbound/HealOverTime version=gms_v84 ida=0xa69c4d
// packet-audit:verify packet=character/serverbound/HealOverTime version=jms_v185 ida=0xb054d6
//
// jms HEAL_OVER_TIME (opcode 0x54) is sent by CWvsContext::SendStatChangeRequestByItemOption@0xb054d6
// (misleading symbol; the opcode is the ground truth — called from CWvsContext::TryRecovery
// auto-recovery). Wire = updateTime(4)+val(4)+hp(2)+mp(2)+option(1)+extra(4); jms appends a
// trailing client validation dword (dword_CDA4F8) the GMS v83/v87/v95 senders do NOT. The
// codec encodes the option byte for (GMS<=95)||JMS and the trailing dword for JMS only.
//
// Legacy GMS (<83) omits the leading updateTime tick entirely (v79 IDA-verified below);
// the round-trip only asserts updateTime for versions that carry it.
func TestHealOverTimeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := HealOverTime{updateTime: 100, val: 200, hp: 50, mp: 30, unknown: 1, extra: 0xCAFEBABE}
			output := HealOverTime{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.HP() != input.HP() {
				t.Errorf("hp: got %v, want %v", output.HP(), input.HP())
			}
			if output.MP() != input.MP() {
				t.Errorf("mp: got %v, want %v", output.MP(), input.MP())
			}
			legacyGMS := v.Region == "GMS" && v.MajorVersion < 83
			if !legacyGMS && output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			} else if legacyGMS && output.UpdateTime() != 0 {
				t.Errorf("legacy GMS must not read an updateTime tick, got %v", output.UpdateTime())
			}
			// jms appends the validation dword (CWvsContext::SendStatChangeRequestByItemOption@0xb054d6,
			// opcode 0x54); GMS does not. Assert the round-trip preserves it only where it is on the wire.
			if v.Region == "JMS" {
				if output.Extra() != input.Extra() {
					t.Errorf("extra (jms trailing dword): got %#x, want %#x", output.Extra(), input.Extra())
				}
			} else if output.Extra() != 0 {
				t.Errorf("extra: GMS must not read a trailing dword, got %#x", output.Extra())
			}
		})
	}
}

// TestHealOverTimeV79ByteOutput pins the gms_v79 HEAL_OVER_TIME (op 0x57) wire.
//
// Sender CWvsContext::SendStatChangeRequest (GMS_v79_1_DEVM.exe @0x96944a):
//
//	COutPacket::COutPacket(v5, 87)   @0x96941c → opcode 87 (matches registry)
//	COutPacket::Encode4(v5, 0x1400)  @0x96942d → val = constant 0x1400
//	COutPacket::Encode2(v5, a2)      @0x969438 → hp (uint16)
//	COutPacket::Encode2(v5, a3)      @0x969443 → mp (uint16)
//	COutPacket::Encode1(v5, a4)      @0x96944e → option byte
//
// There is NO get_update_time call in the v79 sender → no leading updateTime
// dword (unlike v83+). Body = val(4) + hp(2) + mp(2) + option(1) = 9 bytes.
//
// packet-audit:verify packet=character/serverbound/HealOverTime version=gms_v48 ida=0x71a482
// TestHealOverTimeV48ByteOutput pins the gms_v48 HEAL_OVER_TIME (op 68). IDA:
// CWvsContext::SendStatChangeRequest = @0x71a482 (GMS_v48_1_DEVM.exe) builds
// COutPacket(68) then Encode4(0x1400)@0x71a49b + Encode2(hp)@0x71a4a6 +
// Encode2(mp)@0x71a4b1 + Encode1(option)@0x71a4bc — NO leading updateTime tick on
// legacy GMS (<83). Matches the codec's <83 branch (byte-identical to v79).
func TestHealOverTimeV48ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := HealOverTime{updateTime: 100, val: 0x1400, hp: 50, mp: 30, unknown: 1}
	expected := []byte{
		0x00, 0x14, 0x00, 0x00, // val 0x1400 (Encode4)
		0x32, 0x00, // hp 50 (Encode2)
		0x1E, 0x00, // mp 30 (Encode2)
		0x01, // option (Encode1)
	}
	if actual := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(actual, expected) {
		t.Errorf("v48 heal-over-time golden mismatch:\n got %x\nwant %x", actual, expected)
	}
}

// packet-audit:verify packet=character/serverbound/HealOverTime version=gms_v79 ida=0x96940a
func TestHealOverTimeV79ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	// val=0x1400, hp=50, mp=30, option=1. No updateTime prefix on legacy GMS.
	input := HealOverTime{updateTime: 100, val: 0x1400, hp: 50, mp: 30, unknown: 1}
	expected := []byte{
		0x00, 0x14, 0x00, 0x00, // val 0x1400 (Encode4)   /*0x96942d*/
		0x32, 0x00, // hp 50 (Encode2)                    /*0x969438*/
		0x1E, 0x00, // mp 30 (Encode2)                    /*0x969443*/
		0x01, // option (Encode1)                          /*0x96944e*/
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 heal-over-time golden mismatch:\n got %x\nwant %x", actual, expected)
	}
}
