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

// TestSummonRemoveBytesV79 pins the v79 wire byte-for-byte against the live
// decompile (IDA, GMS_v79_1_DEVM.exe @port 13340). v79 is v83-shaped — same
// wire, no version delta. Dispatch chain:
//   - CUserPool::OnUserCommonPacket@0x8c8c79 reads cid (Decode4@0x8c8c84), ops
//     164-169 → summon cluster sub_892500@0x892500; the else branch reads oid
//     (Decode4@0x89253f) then for a2==165 (REMOVE_SPECIAL_MAPOBJECT) calls the
//     OnRemoved leaf sub_71CC52@0x71cc52.
//   - sub_71CC52 reads ONE byte: Decode1@0x71cc67 (leave/animated flag, branched
//     0/2/3/4) and nothing else from the packet.
//
// Wire = int ownerId(=cid, consumed upstream) + int oid + byte flag. Atlas writes
// flag = 4 when animated, else 1 (matches the branch).
// packet-audit:verify packet=summon/clientbound/SummonRemove version=gms_v79 ida=0x71cc52
func TestSummonRemoveBytesV79(t *testing.T) {
	in := NewSummonRemove(42, 1000001, true)
	ctx := test.CreateContext("GMS", 79, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// ownerId=42, oid=1000001=0x000F4241, animated => byte 4 (Decode1@0x71cc67)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // ownerId (cid, consumed by dispatcher)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x89253f in sub_892500)
		0x04, // animated ? 4 : 1 (Decode1@0x71cc67)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 bytes = % X, want % X", got, want)
	}
}

// TestSummonRemoveBytesV72 pins the v72 wire byte-for-byte against the live
// decompile (IDA, GMS_v72.1_U_DEVM.exe @port 13339). v72 is v79/v83-shaped.
// Dispatch chain:
//   - CUserPool::OnUserCommonPacket reads cid; the summon cluster dispatcher
//     sub_848023@0x848023 else branch reads oid (Decode4@0x848062) then for a2==161
//     (REMOVE_SPECIAL_MAPOBJECT) calls the OnRemoved leaf sub_6E8F0F@0x6e8f0f.
//   - sub_6E8F0F reads ONE byte: Decode1@0x6e8f24 (leave/animated flag, branched
//     0/2/3/4) and nothing else from the packet.
//
// Wire = int ownerId(=cid, consumed upstream) + int oid + byte flag.
// packet-audit:verify packet=summon/clientbound/SummonRemove version=gms_v72 ida=0x6e8f0f
func TestSummonRemoveBytesV72(t *testing.T) {
	in := NewSummonRemove(42, 1000001, true)
	ctx := test.CreateContext("GMS", 72, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // ownerId (cid, consumed by dispatcher)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x848062 in sub_848023)
		0x04, // animated ? 4 : 1 (Decode1@0x6e8f24)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 bytes = % X, want % X", got, want)
	}
}

// TestSummonRemoveBytesV83 pins the v83 wire byte-for-byte against the live
// decompile. Dispatch chain (IDA, MapleStory_dump.exe @port 13341):
//   - CUserPool::OnUserCommonPacket@0x972401 reads cid (Decode4@0x97240c), then
//     routes op 0xB0 to CSummonedPool::OnPacket(v6,a2)@0x972490.
//   - CSummonedPool::OnPacket@0x938dd7 reads oid (Decode4@0x938e16), looks up the
//     summon, then for op 0xB0 calls sub_7A64EB(v9,v5)@0x938e43 (the OnRemoved leaf).
//   - sub_7A64EB@0x7a64eb reads ONE byte: Decode1@0x7a6500 (leave/animated flag,
//     branched 0/2/3/4) and nothing else from the packet.
//
// So the wire is: int ownerId(=cid, consumed upstream) + int oid + byte flag.
// Atlas writes flag = 4 when animated, else 1 (matches the branch).
// packet-audit:verify packet=summon/clientbound/SummonRemove version=gms_v83 ida=0x7a64eb
func TestSummonRemoveBytesV83(t *testing.T) {
	in := NewSummonRemove(42, 1000001, true)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// ownerId=42, oid=1000001=0x000F4241, animated => byte 4 (Decode1@0x7a6500)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // ownerId (cid, consumed by dispatcher)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x938e16 in OnPacket)
		0x04, // animated ? 4 : 1 (Decode1@0x7a6500)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v83 bytes = % X, want % X", got, want)
	}
}

// TestSummonRemoveBytesV84 pins the v84 wire byte-for-byte against the live
// decompile (IDA, GMS_v84.1_U_DEVM.exe @port 13337). v84 is v83-shaped — same
// dispatch + leaf, no version delta. Dispatch chain:
//   - CUserPool::OnUserCommonPacket@0x9b23a1 reads cid (Decode4@0x9b23ac), then
//     routes op 0xB4 (180) to the summon dispatcher sub_970201@0x970201.
//   - sub_970201@0x970201 (else branch) reads oid (Decode4@0x970240), looks up the
//     summon via sub_97B9D1, then for op 180 calls the OnRemoved leaf sub_7CBFA1@0x7cbfa1.
//   - sub_7CBFA1@0x7cbfa1 reads ONE byte: Decode1@0x7cbfb6 (leave/animated flag,
//     branched 0/2/3/4) and nothing else from the packet.
//
// Wire = int ownerId(=cid, consumed upstream) + int oid + byte flag. Atlas writes
// flag = 4 when animated, else 1 (matches the branch). No off-by-one: Remove has no
// version gate, so the v84 path is byte-identical to v83.
// packet-audit:verify packet=summon/clientbound/SummonRemove version=gms_v84 ida=0x7cbfa1
func TestSummonRemoveBytesV84(t *testing.T) {
	in := NewSummonRemove(42, 1000001, true)
	ctx := test.CreateContext("GMS", 84, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// ownerId=42, oid=1000001=0x000F4241, animated => byte 4 (Decode1@0x7cbfb6)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // ownerId (cid, consumed by dispatcher)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x970240 in sub_970201)
		0x04, // animated ? 4 : 1 (Decode1@0x7cbfb6)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v84 bytes = % X, want % X", got, want)
	}
}

// TestSummonRemoveBytesV87 pins the v87 wire byte-for-byte against the live
// decompile (IDA, GMSv87_4GB.exe @port 13340). v87 is v83-shaped — same dispatch
// + leaf, no version delta. Dispatch chain:
//   - CUserPool::OnUserCommonPacket@0x9f7387 reads cid (Decode4@0x9f7392), routes
//     ops 188-193 to CSummonedPool::OnPacket@0x9b35bf.
//   - CSummonedPool::OnPacket@0x9b35bf (the non-0xBC arm) reads oid (Decode4@
//     0x9b35fe), looks up the summon via sub_9BEC8B, then for op 0xBD calls the
//     OnRemoved leaf sub_7F8CB0@0x7f8cb0.
//   - sub_7F8CB0@0x7f8cb0 reads ONE byte: Decode1@0x7f8cc5 (leave/animated flag,
//     branched 0/2/3/4) and nothing else from the packet (the rest is local
//     chat/UI logic).
//
// Wire = int ownerId(=cid, consumed upstream) + int oid + byte flag. Atlas writes
// flag = 4 when animated, else 1 (matches the branch). No off-by-one: Remove has no
// version gate, so the v87 path is byte-identical to v83.
// packet-audit:verify packet=summon/clientbound/SummonRemove version=gms_v87 ida=0x7f8cb0
func TestSummonRemoveBytesV87(t *testing.T) {
	in := NewSummonRemove(42, 1000001, true)
	ctx := test.CreateContext("GMS", 87, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// ownerId=42, oid=1000001=0x000F4241, animated => byte 4 (Decode1@0x7f8cc5)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // ownerId (cid, consumed by dispatcher)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x9b35fe in OnPacket)
		0x04, // animated ? 4 : 1 (Decode1@0x7f8cc5)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v87 bytes = % X, want % X", got, want)
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

// TestSummonRemoveBytesJMS185 pins the JMS185 wire byte-for-byte against the live
// decompile (IDA, MapleStory_dump_SCY.exe @port 13338). jms185 is v83-shaped —
// same dispatch + leaf, no version delta. Dispatch chain:
//   - CUserPool::OnUserCommonPacket reads cid, op 0xB6 (182) routes to
//     CSummonedPool::OnPacket@0x9f7f6e, which reads oid (Decode4@0x9f7fad), looks up
//     the summon, then for op 0xB6 calls the OnRemoved leaf sub_828502@0x828502.
//   - sub_828502@0x828502 reads ONE byte: Decode1@0x828517 (v3, the leave/animated
//     flag, branched 0/2/3/4) and nothing else from the packet (the rest is local
//     skill-message / action-layer logic).
//
// Wire = int ownerId(=cid, consumed upstream) + int oid + byte flag. Atlas writes
// flag = 4 when animated, else 1 (matches the branch). Remove has no version gate,
// so the jms185 path is byte-identical to v83.
// packet-audit:verify packet=summon/clientbound/SummonRemove version=jms_v185 ida=0x828502
func TestSummonRemoveBytesJMS185(t *testing.T) {
	in := NewSummonRemove(42, 1000001, true)
	ctx := test.CreateContext("JMS", 185, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// ownerId=42, oid=1000001=0x000F4241, animated => byte 4 (Decode1@0x828517)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // ownerId (cid, consumed by dispatcher)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x9f7fad in OnPacket)
		0x04, // animated ? 4 : 1 (Decode1@0x828517)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("JMS185 bytes = % X, want % X", got, want)
	}
}
