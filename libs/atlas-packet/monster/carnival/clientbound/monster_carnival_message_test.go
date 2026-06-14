package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MONSTER_CARNIVAL_MESSAGE present in all 5 versions — the MESSAGE branch of
// CField_MonsterCarnival::OnRequestResult (dispatcher arg == 0). DISTINCT wire
// shape from MONSTER_CARNIVAL_SUMMON (same source fn, arg != 0): a single byte;
// the displayed text is sourced from the client StringPool, not the packet.
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalMessage version=gms_v83 ida=0x56557d
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalMessage version=gms_v84 ida=0x572284
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalMessage version=gms_v87 ida=0x590303
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalMessage version=gms_v95 ida=0x55a890
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalMessage version=jms_v185 ida=0x5b0332
func TestMonsterCarnivalMessage(t *testing.T) {
	input := NewMonsterCarnivalMessage(3)

	// Golden bytes (v83). OnRequestResult MESSAGE branch (bResult == 0) @0x56557d:
	//   single Decode1 message selector (switch 1..6 -> StringPool string).
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x03, // message selector byte = 3
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterCarnivalMessage layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
