package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterMobAffected version=gms_v83 ida=0x66c675
// packet-audit:verify packet=monster/clientbound/MonsterMobAffected version=gms_v84 ida=0x682977
// packet-audit:verify packet=monster/clientbound/MonsterMobAffected version=gms_v87 ida=0x6a7540
// packet-audit:verify packet=monster/clientbound/MonsterMobAffected version=gms_v95 ida=0x644400
// packet-audit:verify packet=monster/clientbound/MonsterMobAffected version=jms_v185 ida=0x6e9df6
func TestMobAffected(t *testing.T) {
	input := NewMobAffected(0x0103E81B, 0x1234)

	// Golden bytes (v83 baseline). CMob::OnAffected @0x66c675:
	//   v4 = Decode4(a2)  -> skillId int32 LE
	//   v5 = Decode2(a2)  -> delay  uint16 LE  (tStart = v5 + get_update_time())
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x1B, 0xE8, 0x03, 0x01, // skillId int32 LE = 0x0103E81B (Decode4 @0x66c675)
		0x34, 0x12, // delay uint16 LE = 0x1234 (Decode2 @0x66c675)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobAffected layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
