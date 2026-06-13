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
