package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOB_ESCORT_FULL_PATH present in v95 (case 304) + jms (case 0x110). Absent in
// v83/v84/v87 (no escort family; the v87 registry row at 0x111 is a stale csv
// off-by-one with no OnEscortFullPath symbol — removed).
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortFullPath version=gms_v95 ida=0x643d90
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortFullPath version=jms_v185 ida=0x6efa01
func TestMobEscortFullPath(t *testing.T) {
	// Two waypoints, both kind=1 (no per-waypoint extra), arrive present, no reset.
	input := NewMobEscortFullPath(
		0x00000001,
		[]MobEscortWaypoint{
			NewMobEscortWaypoint(0x00000064, 0x000000C8, 1, 0),
			NewMobEscortWaypoint(0x000000FA, 0x0000012C, 1, 0),
		},
		0x00000190,
		true,
		0x00000320,
		false,
	)

	// Golden bytes (v95). CMob::OnEscortFullPath @0x643d90:
	//   Decode4 -> mode
	//   Decode4 -> count
	//   per waypoint: Decode4 x, Decode4 y, Decode4 kind (kind==2 → +Decode4 extra)
	//   Decode4 -> tail
	//   Decode1 -> hasArrive; if set Decode4 -> arriveDelay
	//   Decode1 -> hasReset
	got := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	want := []byte{
		0x01, 0x00, 0x00, 0x00, // mode = 1
		0x02, 0x00, 0x00, 0x00, // count = 2
		0x64, 0x00, 0x00, 0x00, // wp0.x = 100
		0xC8, 0x00, 0x00, 0x00, // wp0.y = 200
		0x01, 0x00, 0x00, 0x00, // wp0.kind = 1
		0xFA, 0x00, 0x00, 0x00, // wp1.x = 250
		0x2C, 0x01, 0x00, 0x00, // wp1.y = 300
		0x01, 0x00, 0x00, 0x00, // wp1.kind = 1
		0x90, 0x01, 0x00, 0x00, // tail = 400
		0x01,                   // hasArrive = true
		0x20, 0x03, 0x00, 0x00, // arriveDelay = 800
		0x00, // hasReset = false
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobEscortFullPath layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
