package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MONSTER_CARNIVAL_START present in all 5 versions (CField_MonsterCarnival::OnEnter).
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalStart version=gms_v83 ida=0x565397
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalStart version=gms_v84 ida=0x57209e
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalStart version=gms_v87 ida=0x59011d
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalStart version=gms_v95 ida=0x55a6c0
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalStart version=jms_v185 ida=0x5b014c
func TestMonsterCarnivalStart(t *testing.T) {
	input := NewMonsterCarnivalStart(1, 0x0102, 0x0304, 0x0506, 0x0708, 0x090A, 0x0B0C, []byte{0x01, 0x00})

	// Golden bytes (v83). CField_MonsterCarnival::OnEnter @0x565397:
	//   Decode1 team, 6x Decode2 (CP totals), then 2x Decode1 (summon-slot spelled bytes).
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x01,       // team byte = 1
		0x02, 0x01, // personalCp uint16 LE = 0x0102
		0x04, 0x03, // personalTotal uint16 LE = 0x0304
		0x06, 0x05, // myTeamCp uint16 LE = 0x0506
		0x08, 0x07, // myTeamTotal uint16 LE = 0x0708
		0x0A, 0x09, // enemyTeamCp uint16 LE = 0x090A
		0x0C, 0x0B, // enemyTeamTotal uint16 LE = 0x0B0C
		0x01, 0x00, // spelled[0..1] (one Decode1 per summon slot)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterCarnivalStart layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			in := NewMonsterCarnivalStart(1, 0x0102, 0x0304, 0x0506, 0x0708, 0x090A, 0x0B0C, []byte{0x01, 0x00})
			pt.RoundTrip(t, ctx, in.Encode, in.Decode, nil)
		})
	}
}
