package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MONSTER_CARNIVAL_PARTY_CP present in all 5 versions (CField_MonsterCarnival::OnTeamCP).
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalPartyCP version=gms_v83 ida=0x56553e
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalPartyCP version=gms_v84 ida=0x572245
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalPartyCP version=gms_v87 ida=0x5902c4
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalPartyCP version=gms_v95 ida=0x55a2d0
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalPartyCP version=jms_v185 ida=0x5b02f3
func TestMonsterCarnivalPartyCP(t *testing.T) {
	input := NewMonsterCarnivalPartyCP(1, 0x0102, 0x0304)

	// Golden bytes (v83). CField_MonsterCarnival::OnTeamCP @0x56553e:
	//   Decode1 team, Decode2 cp, Decode2 total -> SetTeamCP(team, cp, total).
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x01,       // team byte = 1
		0x02, 0x01, // cp uint16 LE = 0x0102
		0x04, 0x03, // total uint16 LE = 0x0304
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterCarnivalPartyCP layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
