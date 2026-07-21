package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MONSTER_CARNIVAL_LEAVE present in all 5 versions (CField_MonsterCarnival::OnShowMemberOutMsg).
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalLeave version=gms_v79 ida=0x5488ef
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalLeave version=gms_v83 ida=0x565962
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalLeave version=gms_v84 ida=0x572669
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalLeave version=gms_v87 ida=0x5906e3
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalLeave version=gms_v95 ida=0x55ad80
// packet-audit:verify packet=monster/carnival/clientbound/MonsterCarnivalLeave version=jms_v185 ida=0x5b070f
func TestMonsterCarnivalLeave(t *testing.T) {
	input := NewMonsterCarnivalLeave(6, 1, "Hero")

	// Golden bytes (v83). CField_MonsterCarnival::OnShowMemberOutMsg @0x565962:
	//   Decode1 leader (==6 => leader variant), Decode1 team, DecodeStr name.
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x06,       // leader byte = 6
		0x01,       // team byte = 1
		0x04, 0x00, // name length uint16 LE = 4
		0x48, 0x65, 0x72, 0x6F, // "Hero"
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterCarnivalLeave layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMonsterCarnivalLeaveByteOutputV79 pins the gms_v79 wire bytes.
// CField_MonsterCarnival::OnShowMemberOutMsg @0x5488ef (v79): Decode1 leader
// (==6 => leader variant), Decode1 team, DecodeStr name — identical read
// order to v83/v84/v87/v95/jms.
func TestMonsterCarnivalLeaveByteOutputV79(t *testing.T) {
	input := NewMonsterCarnivalLeave(6, 1, "Hero")

	got := input.Encode(nil, pt.CreateContext("GMS", 79, 1))(nil)
	want := []byte{
		0x06,       // leader byte = 6
		0x01,       // team byte = 1
		0x04, 0x00, // name length uint16 LE = 4
		0x48, 0x65, 0x72, 0x6F, // "Hero"
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterCarnivalLeave v79 layout mismatch\n got % x\nwant % x", got, want)
	}

	ctx := pt.CreateContext("GMS", 79, 1)
	pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
}
