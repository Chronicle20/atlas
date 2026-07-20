package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MONSTER_CARNIVAL_RESULT present in all 5 versions (CField_MonsterCarnival::OnShowGameResult).
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalResult version=gms_v79 ida=0x548a6a
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalResult version=gms_v83 ida=0x565add
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalResult version=gms_v84 ida=0x5727e4
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalResult version=gms_v87 ida=0x59085e
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalResult version=gms_v95 ida=0x55af80
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalResult version=jms_v185 ida=0x5b088a
func TestMonsterCarnivalResult(t *testing.T) {
	input := NewMonsterCarnivalResult(8)

	// Golden bytes (v83). CField_MonsterCarnival::OnShowGameResult @0x565add:
	//   single Decode1 result selector (8=win, 9=lose, 10=draw, 11=opponent left).
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x08, // result selector byte = 8 (win)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterCarnivalResult layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMonsterCarnivalResultByteOutputV79 pins the gms_v79
// MONSTER_CARNIVAL_RESULT clientbound read. IDA:
// CField_MonsterCarnival::OnShowGameResult @0x548a6a (GMS_v79_1_DEVM.exe)
// reads a single Decode1 result selector. Body is byte-identical to the v83
// golden.
func TestMonsterCarnivalResultByteOutputV79(t *testing.T) {
	input := NewMonsterCarnivalResult(8)
	got := input.Encode(nil, pt.CreateContext("GMS", 79, 1))(nil)
	want := []byte{
		0x08, // result selector byte = 8 (win)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterCarnivalResult v79 layout mismatch\n got % x\nwant % x", got, want)
	}
}
