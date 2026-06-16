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

// TestSummonRemoveBytes pins the v83 wire: ownerId + oid + animated byte. The cid
// is read upstream by CUserPool::OnUserCommonPacket@0x972401; CSummonedPool::
// OnPacket@0x938dd7 then does one Decode4 = the oid before the pool-remove
// (sub_7A64EB). (The prior "no oid" reading missed the upstream cid — see
// summon-wire-truth.md.) NOTE: v84/v87/jms inherit this correction; their matrix
// cells need re-verification against the cid-pre-reading dispatcher.
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
