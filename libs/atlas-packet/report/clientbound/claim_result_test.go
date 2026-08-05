package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestClaimResultSuccessGolden(t *testing.T) {
	// mode 2 = success: byte hasRemaining, int32 remaining ("D reports left this week").
	input := NewClaimResultSuccess(0x02, true, 100)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x02, 0x01, 0x64, 0x00, 0x00, 0x00}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimResultNoticeGolden(t *testing.T) {
	// mode 0x42 = "Please re-check the character name then try again" — bare mode byte.
	input := NewClaimResultNotice(0x42)
	ctx := pt.CreateContext("GMS", 95, 1)
	expected := []byte{0x42}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimResultSuccessByteOutputV72 verifies the wire-exact byte output of
// ClaimResultSuccess (mode 2) for GMS v72.
// IDA evidence (session c8acae95, GMS_v72.1_U_DEVM.exe.i64):
//
//	CWvsContext::OnClaimResult@0x91f9a9, resolved via the opcode dispatch table
//	CWvsContext::OnPacket case 42 @0x90271c (registry gms_v72.yaml op
//	CLAIM_RESULT, opcode 42/0x2A). CInPacket::Decode1(a2) @0x91f9be reads the
//	mode byte (v3). mode==2 is the ONLY value that reads further: Decode1(a2)
//	@0x91fa3d reads hasRemaining (v8), Decode4(a2) @0x91fa47 reads remaining
//	(a2, reused as an int32). Every other reachable mode (3, 0x41-0x45, 0x47,
//	0x48 -- see the registry note on this op) is a bare mode byte with no
//	further packet reads -- ClaimResultNotice, verified below. 6 bytes total
//	for mode 2: mode, hasRemaining, remaining(4).
//
// packet-audit:verify packet=report/clientbound/ClaimResultSuccess version=gms_v72 ida=0x91f9a9
func TestClaimResultSuccessByteOutputV72(t *testing.T) {
	v := pt.Variants[9] // GMS v72
	if v.Name != "GMS v72" {
		t.Fatalf("pt.Variants[9] = %q, want %q (index drifted)", v.Name, "GMS v72")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimResultSuccess(0x02, true, 100)
	expected := []byte{
		0x02,                   // Decode1 mode @0x91f9be
		0x01,                   // Decode1 hasRemaining @0x91fa3d
		0x64, 0x00, 0x00, 0x00, // Decode4 remaining = 100 LE @0x91fa47
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimResultNoticeByteOutputV72 verifies the wire-exact byte output of
// ClaimResultNotice (bare mode byte, no payload) for GMS v72, covering every
// reachable non-success mode.
// IDA evidence (session c8acae95, GMS_v72.1_U_DEVM.exe.i64), all within
// CWvsContext::OnClaimResult@0x91f9a9 after the single Decode1(mode)
// @0x91f9be -- none of these branches perform any further CInPacket read:
//
//	mode 3    -> StringPool 3359 @0x91fa25
//	mode 0x41 -> StringPool 3361 @0x91fa0f
//	mode 0x42 -> StringPool 3362 @0x91f9f9
//	mode 0x43 -> StringPool 3363 @0x91fb57
//	mode 0x44 -> StringPool 3364 @0x91fc2b
//	mode 0x45 -> StringPool 3365 @0x91fc17
//	mode 0x47 -> StringPool 3370 @0x91fba2 (own CHATLOG_ADD+Notice arm)
//	mode 0x48 -> StringPool 3372 @0x91fb88
//
// (mode 0x46 and every other unlisted byte value fall through to an early
// return with no display -- still just the 1-byte mode already consumed.)
// Not registry-linked (the CLAIM_RESULT op row's single `packet:` field
// points at ClaimResultSuccess above) so this test carries no verify-marker
// comment, but the decompile citations are load-bearing for the registry
// note on CLAIM_RESULT.
func TestClaimResultNoticeByteOutputV72(t *testing.T) {
	v := pt.Variants[9] // GMS v72
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	for _, mode := range []byte{0x03, 0x41, 0x42, 0x43, 0x44, 0x45, 0x47, 0x48} {
		input := NewClaimResultNotice(mode)
		expected := []byte{mode}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("mode 0x%02X: byte output mismatch: got %v want %v", mode, actual, expected)
		}
	}
}

func TestClaimResultSuccessRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClaimResultSuccess(0x02, true, 42)
			output := ClaimResultSuccess{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() || output.HasRemaining() != input.HasRemaining() || output.Remaining() != input.Remaining() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}

func TestClaimResultNoticeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClaimResultNotice(0x41)
			output := ClaimResultNotice{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("round-trip mismatch: got %d want %d", output.Mode(), input.Mode())
			}
		})
	}
}
