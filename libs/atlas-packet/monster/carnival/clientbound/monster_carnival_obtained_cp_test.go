package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MONSTER_CARNIVAL_OBTAINED_CP present in all 5 versions (CField_MonsterCarnival::OnPersonalCP).
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalObtainedCP version=gms_v83 ida=0x56550e
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalObtainedCP version=gms_v84 ida=0x572215
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalObtainedCP version=gms_v87 ida=0x590294
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalObtainedCP version=gms_v95 ida=0x55a2a0
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalObtainedCP version=jms_v185 ida=0x5b02c3
func TestMonsterCarnivalObtainedCP(t *testing.T) {
	input := NewMonsterCarnivalObtainedCP(0x0102, 0x0304)

	// Golden bytes (v83). CField_MonsterCarnival::OnPersonalCP @0x56550e:
	//   Decode2 cp, Decode2 total -> SetPersonalCP(cp, total).
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x02, 0x01, // cp uint16 LE = 0x0102
		0x04, 0x03, // total uint16 LE = 0x0304
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterCarnivalObtainedCP layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
