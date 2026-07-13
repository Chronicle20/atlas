package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/BuffCancelRequest version=gms_v83 ida=0x96d873
// packet-audit:verify packet=character/serverbound/BuffCancelRequest version=gms_v87 ida=0x9f22b8
// packet-audit:verify packet=character/serverbound/BuffCancelRequest version=gms_v95 ida=0x93d730
// packet-audit:verify packet=character/serverbound/BuffCancelRequest version=gms_v84 ida=0x9ad694
// packet-audit:verify packet=character/serverbound/BuffCancelRequest version=jms_v185 ida=0xa3e3ec
func TestBuffCancelRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BuffCancelRequest{skillId: 1001003}
			output := BuffCancelRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SkillId() != input.SkillId() {
				t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
			}
		})
	}
}

// TestBuffCancelRequestBytesV48 pins the very-legacy GMS v48 CANCEL_BUFF (op 71)
// serverbound wire: a single Int32 skillId, version-independent. Body-verified:
// CUserLocal::SendSkillCancelRequest @0x6afcba (GMS_v48_1_DEVM.exe, port 13337) builds
// COutPacket(71) @0x6afcf0 then Encode4(skillId) @0x6afcfd — nothing else. Opcode 71
// confirmed at the send-site (distrust symbols).
// packet-audit:verify packet=character/serverbound/BuffCancelRequest version=gms_v48 ida=0x6afcba
func TestBuffCancelRequestBytesV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := BuffCancelRequest{skillId: 1001003}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x2B, 0x46, 0x0F, 0x00, // skillId=1001003 LE  @0x6afcfd
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 BuffCancelRequest bytes:\n got=% X\nwant=% X", got, want)
	}
}
