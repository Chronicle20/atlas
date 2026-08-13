package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestMobAffectedBytesV48 — CMob::OnAffected @0x55114c reads exactly one Decode4
// (skillId) then one Decode2 (delay), matching every version already verified for
// this codec. The mob id is consumed one level up by CMobPool::OnMobPacket
// @0x559390 (Decode4 @0x55939a) before it switches to case 168 for OnAffected;
// v61 routes the same leaf at 184.
//
// packet-audit:verify packet=monster/clientbound/MonsterMobAffected version=gms_v48 ida=0x55114c
func TestMobAffectedBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := NewMobAffected(2001, 300).Encode(l, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0xD1, 0x07, 0x00, 0x00, // skillId 2001 — Decode4
		0x2C, 0x01, // delay 300    — Decode2
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 mob affected:\n got % x\nwant % x", got, want)
	}
}
