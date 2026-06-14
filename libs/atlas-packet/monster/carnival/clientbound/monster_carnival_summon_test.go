package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MONSTER_CARNIVAL_SUMMON present in all 5 versions — the SUMMON branch of
// CField_MonsterCarnival::OnRequestResult (dispatcher arg != 0). DISTINCT wire
// shape from MONSTER_CARNIVAL_MESSAGE (same source fn, arg == 0).
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalSummon version=gms_v83 ida=0x56557d
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalSummon version=gms_v84 ida=0x572284
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalSummon version=gms_v87 ida=0x590303
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalSummon version=gms_v95 ida=0x55a890
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalSummon version=jms_v185 ida=0x5b0332
func TestMonsterCarnivalSummon(t *testing.T) {
	input := NewMonsterCarnivalSummon(1, 2, "Hero")

	// Golden bytes (v83). OnRequestResult SUMMON branch (bResult != 0) @0x56557d:
	//   Decode1 tab, Decode1 idx, DecodeStr name -> RequestResult(tab, idx, name).
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x01,       // tab byte = 1
		0x02,       // idx byte = 2
		0x04, 0x00, // name length uint16 LE = 4
		0x48, 0x65, 0x72, 0x6F, // "Hero"
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterCarnivalSummon layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
