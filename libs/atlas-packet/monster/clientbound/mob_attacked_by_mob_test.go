package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOB_ATTACKED_BY_MOB present in all five (dispatcher cases 0xFF/262/0x110/309/0x114).
// packet-audit:verify packet=monster/clientbound/MonsterMobAttackedByMob version=gms_v83 ida=0x670f41
// packet-audit:verify packet=monster/clientbound/MonsterMobAttackedByMob version=gms_v84 ida=0x68749a
// packet-audit:verify packet=monster/clientbound/MonsterMobAttackedByMob version=gms_v87 ida=0x6ac074
// packet-audit:verify packet=monster/clientbound/MonsterMobAttackedByMob version=gms_v95 ida=0x6436a0
// packet-audit:verify packet=monster/clientbound/MonsterMobAttackedByMob version=jms_v185 ida=0x6ee151
func TestMobAttackedByMob(t *testing.T) {
	// attackIndex >= 0 → the >-2 branch is taken; full 4-field on-wire form.
	input := NewMobAttackedByMob(0x02, 0x000004D2, 0x000186A0, true)

	// Golden bytes (v95). CMob::OnMobAttackedByMob @0x6436a0:
	//   Decode1 -> attackIndex int8
	//   Decode4 -> damage int32 LE (ShowDamage)
	//   if (attackIndex > -2): Decode4 -> mobTemplateId int32 LE; Decode1 -> left bool
	got := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	want := []byte{
		0x02,                   // attackIndex int8 = 2
		0xD2, 0x04, 0x00, 0x00, // damage int32 LE = 1234
		0xA0, 0x86, 0x01, 0x00, // mobTemplateId int32 LE = 100000
		0x01, // left bool = true
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobAttackedByMob layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
