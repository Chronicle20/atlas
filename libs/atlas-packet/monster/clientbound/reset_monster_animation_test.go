package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterResetMonsterAnimation version=gms_v83 ida=0x66c500
// packet-audit:verify packet=monster/clientbound/MonsterResetMonsterAnimation version=gms_v84 ida=0x682802
// packet-audit:verify packet=monster/clientbound/MonsterResetMonsterAnimation version=gms_v87 ida=0x6a73cb
// packet-audit:verify packet=monster/clientbound/MonsterResetMonsterAnimation version=gms_v95 ida=0x64acb0
// packet-audit:verify packet=monster/clientbound/MonsterResetMonsterAnimation version=jms_v185 ida=0x6e9c8d
func TestResetMonsterAnimation(t *testing.T) {
	input := NewResetMonsterAnimation(true)

	// Golden bytes (v83 baseline). CMob::OnSuspendReset @0x66c500 reads exactly one
	// CInPacket::Decode1 that gates the whole reset body — a single bool byte.
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x01, // animate bool = true (Decode1 @0x66c500)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("ResetMonsterAnimation layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
