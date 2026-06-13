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

// TestSummonRemoveBytes pins the classic (pre-95) wire: ownerId + animated byte,
// NO oid (the remove path sub_7A64EB keys off the dispatcher-consumed cid; oid
// is a v95+ addition — IDB-confirmed, summon-wire-truth.md). v87 is byte-identical:
// the op 0xBD remove path sub_7F8CB0@0x7f8cb0 reads one Decode1 (animated flag),
// no oid. v84 is byte-identical: the field op 0xB4 remove path sub_7CBFA1@0x7cbfa1
// reads one Decode1 (leave/animated flag) after the dispatcher-consumed cid, no
// oid (GMS_v84.1 IDB-confirmed).
// packet-audit:verify packet=summon/clientbound/SummonRemove version=gms_v83 ida=0x7a64eb
// packet-audit:verify packet=summon/clientbound/SummonRemove version=gms_v87 ida=0x7f8cb0
// packet-audit:verify packet=summon/clientbound/SummonRemove version=gms_v84 ida=0x7cbfa1
func TestSummonRemoveBytes(t *testing.T) {
	in := NewSummonRemove(42, 1000001, true)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// ownerId=42, animated => byte 4 (no oid)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // ownerId
		0x04, // animated ? 4 : 1
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v83 bytes = % X, want % X", got, want)
	}
}

// TestSummonRemoveBytesV95 pins the v95+ DELTA: the oid int after ownerId.
// packet-audit:verify packet=summon/clientbound/SummonRemove version=gms_v95 ida=0x75a470
func TestSummonRemoveBytesV95(t *testing.T) {
	in := NewSummonRemove(42, 1000001, true)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// ownerId=42, oid=1000001=0x000F4241, animated => byte 4
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // ownerId
		0x41, 0x42, 0x0F, 0x00, // oid (v95+ only)
		0x04, // animated ? 4 : 1
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 bytes = % X, want % X", got, want)
	}
}
