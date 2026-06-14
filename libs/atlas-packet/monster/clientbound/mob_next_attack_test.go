package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOB_NEXT_ATTACK is v95-only (dispatcher case 308 @0x6570b0). Absent in
// v83/v84/v87/jms (no NextAttack dispatcher case / symbol).
// packet-audit:verify packet=monster/clientbound/MonsterMobNextAttack version=gms_v95 ida=0x6528a0
func TestMobNextAttack(t *testing.T) {
	input := NewMobNextAttack(0x00000004)

	// Golden bytes (v95). CMob::OnNextAttack @0x6528a0:
	//   Decode4 -> attackId int32 LE (IsTargetInAttackRange / GenerateMovePath)
	got := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	want := []byte{
		0x04, 0x00, 0x00, 0x00, // attackId int32 LE = 4
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobNextAttack layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
