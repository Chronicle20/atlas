package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOB_ESCORT_STOP_END_REQUEST present in v95 (opcode 238) + jms (opcode 0xCD).
// Absent in v83/v84/v87 (no escort family).
// packet-audit:verify packet=monster/serverbound/MonsterMobEscortStopEndRequest version=gms_v95 ida=0x641290
// packet-audit:verify packet=monster/serverbound/MonsterMobEscortStopEndRequest version=jms_v185 ida=0x6effcd
func TestMobEscortStopEndRequest(t *testing.T) {
	input := MobEscortStopEndRequest{mobCrc: 0x55667788}

	// Golden bytes (v95). CMob::SendEscortStopEndRequest @0x641290:
	//   Encode4(SecureFuse(m_dwMobID)) -> mobCrc uint32 LE
	got := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	want := []byte{
		0x88, 0x77, 0x66, 0x55, // mobCrc uint32 LE = 0x55667788
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobEscortStopEndRequest layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
