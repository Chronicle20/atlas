package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOB_REQUEST_ESCORT_INFO present in v95 (opcode 237) + jms (opcode 0xCC). Absent
// in v83/v84/v87 (no escort family).
// packet-audit:verify packet=monster/serverbound/MonsterMobRequestEscortInfo version=gms_v95 ida=0x6411f0
// packet-audit:verify packet=monster/serverbound/MonsterMobRequestEscortInfo version=jms_v185 ida=0x6eff57
func TestMobRequestEscortInfo(t *testing.T) {
	input := MobRequestEscortInfo{mobCrc: 0x11223344}

	// Golden bytes (v95). CMob::SendRequestEscortPath @0x6411f0:
	//   Encode4(SecureFuse(m_dwMobID)) -> mobCrc uint32 LE
	got := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	want := []byte{
		0x44, 0x33, 0x22, 0x11, // mobCrc uint32 LE = 0x11223344
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobRequestEscortInfo layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
