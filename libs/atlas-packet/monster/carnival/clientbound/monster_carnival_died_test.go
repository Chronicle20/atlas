package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MONSTER_CARNIVAL_DIED present in all 5 versions (CField_MonsterCarnival::OnProcessForDeath).
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalDied version=gms_v79 ida=0x548774
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalDied version=gms_v83 ida=0x5657e7
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalDied version=gms_v84 ida=0x5724ee
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalDied version=gms_v87 ida=0x590568
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalDied version=gms_v95 ida=0x55ab90
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalDied version=jms_v185 ida=0x5b0597
func TestMonsterCarnivalDied(t *testing.T) {
	input := NewMonsterCarnivalDied(1, "Hero", 5)

	// Golden bytes (v83). CField_MonsterCarnival::OnProcessForDeath @0x5657e7:
	//   Decode1 team, DecodeStr name, Decode1 lostCp.
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x01,       // team byte = 1
		0x04, 0x00, // name length uint16 LE = 4
		0x48, 0x65, 0x72, 0x6F, // "Hero"
		0x05, // lostCp byte = 5
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterCarnivalDied layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMonsterCarnivalDiedByteOutputV79 pins the gms_v79 wire bytes.
// CField_MonsterCarnival::OnProcessForDeath @0x548774 (v79): Decode1 team,
// DecodeStr name, Decode1 lostCp — identical read order to v83/v84/v87/v95/jms.
func TestMonsterCarnivalDiedByteOutputV79(t *testing.T) {
	input := NewMonsterCarnivalDied(1, "Hero", 5)

	got := input.Encode(nil, pt.CreateContext("GMS", 79, 1))(nil)
	want := []byte{
		0x01,       // team byte = 1
		0x04, 0x00, // name length uint16 LE = 4
		0x48, 0x65, 0x72, 0x6F, // "Hero"
		0x05, // lostCp byte = 5
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterCarnivalDied v79 layout mismatch\n got % x\nwant % x", got, want)
	}

	ctx := pt.CreateContext("GMS", 79, 1)
	pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
}
