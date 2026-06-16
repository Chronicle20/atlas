package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOB_ESCORT_RETURN_BEFORE present in v95 (case 307) + jms (case 0x113). Absent in
// v83/v84/v87 (no escort family).
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortReturnBefore version=gms_v95 ida=0x649410
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortReturnBefore version=jms_v185 ida=0x6f029c
func TestMobEscortReturnBefore(t *testing.T) {
	input := NewMobEscortReturnBefore(0x00000003)

	// Golden bytes (v95). CMob::OnEscortReturnBefore @0x649410:
	//   Decode4 -> index int32 LE (escort waypoint to return before)
	got := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	want := []byte{
		0x03, 0x00, 0x00, 0x00, // index int32 LE = 3
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobEscortReturnBefore layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
