package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestSummonDamage(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, in.Encode, in.Decode, nil)
		})
	}
}

// TestSummonDamageBytes pins the exact wire layout. SummonDamage is
// byte-identical across all versions (summon-packet-delta.md §3.5), so a single
// v83 assertion guards against an accidental version branch.
// packet-audit:verify packet=summon/clientbound/SummonDamage version=gms_v95 ida=0x7598c0
func TestSummonDamageBytes(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=0x000F4241, byte 12, damage=1234=0x000004D2,
	// monsterIdFrom=9300018=0x008DE832, byte 0
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid
		0x41, 0x42, 0x0F, 0x00, // oid
		0x0C,                   // byte 12
		0xD2, 0x04, 0x00, 0x00, // damage
		0x32, 0xE8, 0x8D, 0x00, // monsterIdFrom
		0x00, // byte 0
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("bytes = % X, want % X", got, want)
	}
}
