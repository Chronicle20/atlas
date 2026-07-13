package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/serverbound/MonsterMobCrcKeyChangedReply version=gms_v83 ida=0x6797be
// packet-audit:verify packet=monster/serverbound/MonsterMobCrcKeyChangedReply version=gms_v84 ida=0x690354
// packet-audit:verify packet=monster/serverbound/MonsterMobCrcKeyChangedReply version=gms_v87 ida=0x6b5399
// packet-audit:verify packet=monster/serverbound/MonsterMobCrcKeyChangedReply version=gms_v95 ida=0x657230
// packet-audit:verify packet=monster/serverbound/MonsterMobCrcKeyChangedReply version=jms_v185 ida=0x6f8bcb
func TestMobCrcKeyChangedReply(t *testing.T) {
	input := MobCrcKeyChangedReply{}

	// Golden bytes (v83 baseline): EMPTY. CMobPool::OnMobCrcKeyChanged @0x6797be
	// builds the reply COutPacket and SendPacket()s with zero Encode* calls.
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobCrcKeyChangedReply layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMobCrcKeyChangedReplyBytesV79 pins the v79 wire: EMPTY payload. The reply
// is built inside CMobPool::OnMobCrcKeyChanged @0x647197 (GMS_v79_1_DEVM.exe,
// port 13340): after Decode4(crcKey) and the mob-list re-checksum loop it does
// COutPacket(154) @0x6471ee then SendPacket @0x647201 with zero Encode* calls.
// Byte-identical to v83; no codec change.
//
// packet-audit:verify packet=monster/serverbound/MonsterMobCrcKeyChangedReply version=gms_v79 ida=0x647197
func TestMobCrcKeyChangedReplyBytesV79(t *testing.T) {
	input := MobCrcKeyChangedReply{}
	ctx := pt.CreateContext("GMS", 79, 1)
	want := []byte{}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 mobCrcKeyChangedReply bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMobCrcKeyChangedReplyBytesV72 pins the v72 wire: EMPTY payload. The reply
// is built inside CMobPool::OnMobCrcKeyChanged @0x625a42 (GMS_v72.1_U_DEVM.exe,
// port 13339): after Decode4(crcKey) @0x625a5a and the mob-list re-checksum loop
// it does COutPacket(155) @0x625a99 then SendPacket @0x625aac with zero Encode*
// calls. Opcode is 155 in v72 (v79 used 154). Empty body — byte-identical shape.
//
// packet-audit:verify packet=monster/serverbound/MonsterMobCrcKeyChangedReply version=gms_v72 ida=0x625a42
func TestMobCrcKeyChangedReplyBytesV72(t *testing.T) {
	input := MobCrcKeyChangedReply{}
	ctx := pt.CreateContext("GMS", 72, 1)
	want := []byte{}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v72 mobCrcKeyChangedReply bytes:\n got % x\nwant % x", got, want)
	}
}
