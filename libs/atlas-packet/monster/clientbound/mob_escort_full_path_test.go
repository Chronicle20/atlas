package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOB_ESCORT_FULL_PATH is routed at gms_v92 (0x128), gms_v95 (0x130) and
// jms_v185 (0x110). Absent in v83/v84/v87 (no escort family). The wire layout is
// byte-identical across all three routed versions — no version gate.
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortFullPath version=gms_v92 ida=0x6374c0
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortFullPath version=gms_v95 ida=0x643d90
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortFullPath version=jms_v185 ida=0x6efa01
func TestMobEscortFullPath(t *testing.T) {
	// Two waypoints: wp0 attr=1 (no per-waypoint stopDuration), wp1 attr=2 (one
	// follows). Escort stop-duration present; no indefinite stop.
	input := NewMobEscortFullPath(
		0x0000000A, // oldDestX
		0x00000014, // oldDestY
		[]MobEscortWaypoint{
			NewMobEscortWaypoint(0x00000064, 0x000000C8, 1, 0),
			NewMobEscortWaypoint(0x000000FA, 0x0000012C, 2, 0x000002BC),
		},
		0x00000001, // currentDestIndex
		true,       // hasStopDuration
		0x00000320, // stopDuration
		false,      // stopIndefinitely
	)

	// Golden bytes. CMob::OnEscortFullPath — v92 @0x6374c0, v95 @0x643d90,
	// jms @0x6efa01; identical read order in all three:
	//   Decode4 -> count (ZArray<CVecCtrlMob::EscortDest>::_Alloc bound)
	//   Decode4 -> m_Old_Dest.dp.x
	//   Decode4 -> m_Old_Dest.dp.y
	//   per waypoint: Decode4 x, Decode4 y, Decode4 nAttr
	//                 (nAttr==2 -> +Decode4 nStopDuration)
	//   Decode4 -> m_nCurrentDestIndex
	//   Decode1 -> hasStopDuration; if set Decode4 -> m_nStopDuration
	//   Decode1 -> stopIndefinitely (m_bEscortStop, no auto-resume)
	got := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	want := []byte{
		0x02, 0x00, 0x00, 0x00, // count = 2
		0x0A, 0x00, 0x00, 0x00, // oldDestX = 10
		0x14, 0x00, 0x00, 0x00, // oldDestY = 20
		0x64, 0x00, 0x00, 0x00, // wp0.x = 100
		0xC8, 0x00, 0x00, 0x00, // wp0.y = 200
		0x01, 0x00, 0x00, 0x00, // wp0.attr = 1
		0xFA, 0x00, 0x00, 0x00, // wp1.x = 250
		0x2C, 0x01, 0x00, 0x00, // wp1.y = 300
		0x02, 0x00, 0x00, 0x00, // wp1.attr = 2
		0xBC, 0x02, 0x00, 0x00, // wp1.stopDuration = 700 (present: attr == 2)
		0x01, 0x00, 0x00, 0x00, // currentDestIndex = 1
		0x01,                   // hasStopDuration = true
		0x20, 0x03, 0x00, 0x00, // stopDuration = 800
		0x00, // stopIndefinitely = false
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
