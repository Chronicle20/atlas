package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MONSTER_CARNIVAL present in all 5 versions (CUIMonsterCarnival::RequestSend).
// packet-audit:verify packet=monster/carnival/serverbound/MonsterCarnival version=gms_v83 ida=0x8706d3
// packet-audit:verify packet=monster/carnival/serverbound/MonsterCarnival version=gms_v84 ida=0x89bdda
// packet-audit:verify packet=monster/carnival/serverbound/MonsterCarnival version=gms_v87 ida=0x8d93c3
// packet-audit:verify packet=monster/carnival/serverbound/MonsterCarnival version=gms_v95 ida=0x80b4a0
// packet-audit:verify packet=monster/carnival/serverbound/MonsterCarnival version=jms_v185 ida=0x903e24
func TestMonsterCarnival(t *testing.T) {
	input := NewMonsterCarnival(2, 0x00000041)

	// Golden bytes (v83). CUIMonsterCarnival::RequestSend @0x8706d3:
	//   Encode1(m_nCurTab), Encode4(m_dwCurIdx - 1). idx carries the already-decremented value.
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x02,                   // tab byte = 2
		0x41, 0x00, 0x00, 0x00, // idx int32 LE = 0x41
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterCarnival layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
