package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOB_ESCORT_STOP present in v95 (case 305, registry row). jms dispatches case 273
// to the same handler but carries no registry row (reported gap). Absent in
// v83/v84/v87 (no escort family). v95 marker only.
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortStop version=gms_v95 ida=0x63b9c0
func TestMobEscortStop(t *testing.T) {
	input := MobEscortStop{}

	// Golden bytes (v95). CMob::OnEscortStopEndPermmision @0x63b9c0 takes NO
	// CInPacket and reads nothing — empty payload (opcode + mob oid only).
	got := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	want := []byte{}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobEscortStop layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
