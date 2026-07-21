package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MONSTER_CARNIVAL_SUMMON present in all 5 versions — the SUMMON branch of
// CField_MonsterCarnival::OnRequestResult (dispatcher arg != 0). DISTINCT wire
// shape from MONSTER_CARNIVAL_MESSAGE (same source fn, arg == 0).
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalSummon version=gms_v79 ida=0x54850a
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

// TestMonsterCarnivalSummonByteOutputV79 pins the gms_v79 MONSTER_CARNIVAL_SUMMON
// clientbound read. IDA: CField_MonsterCarnival::OnRequestResult @0x54850a
// (GMS_v79_1_DEVM.exe), the arg!=0 (SUMMON) branch confirmed against the live
// OnPacket dispatcher switch (case 270 -> OnRequestResult(1, packet)): Decode1
// tab, Decode1 idx, DecodeStr name -> RequestResult(tab, idx, name).
// Byte-identical to v83/v84/v87/v95/jms — the pre-existing atlas codec already
// modelled this shape correctly; this was a route-only gap (v79 opCode 0x10E
// was unrouted, v79 export was unresolved).
func TestMonsterCarnivalSummonByteOutputV79(t *testing.T) {
	input := NewMonsterCarnivalSummon(1, 2, "Hero")
	ctx := pt.CreateContext("GMS", 79, 1)
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x01,       // tab byte = 1
		0x02,       // idx byte = 2
		0x04, 0x00, // name length uint16 LE = 4
		0x48, 0x65, 0x72, 0x6F, // "Hero"
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 MonsterCarnivalSummon layout mismatch\n got % x\nwant % x", got, want)
	}
}
