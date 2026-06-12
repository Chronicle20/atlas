package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestSummonRemove(t *testing.T) {
	in := NewSummonRemove(42, 1000001, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, in.Encode, in.Decode, nil)
		})
	}
}

// TestSummonRemoveBytes pins the exact wire layout. SummonRemove is
// byte-identical across all versions (summon-packet-delta.md §3.2), so a single
// v83 assertion guards against an accidental version branch.
func TestSummonRemoveBytes(t *testing.T) {
	in := NewSummonRemove(42, 1000001, true)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// ownerId=42, oid=1000001=0x000F4241, animated => byte 4
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // ownerId
		0x41, 0x42, 0x0F, 0x00, // oid
		0x04, // animated ? 4 : 1
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("bytes = % X, want % X", got, want)
	}
}
