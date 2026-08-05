package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestSueCharacterResultGolden(t *testing.T) {
	// 1 byte result code; 1 = "Unable to locate the user" (packet-findings.md §1).
	input := NewSueCharacterResult(0x01)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestSueCharacterResultByteOutputV61 verifies the wire-exact byte output of
// SueCharacterResult for GMS v61.
// IDA evidence (session 415bf585, GMS_v61.1_U_DEVM.exe.i64):
//
//	v61 CWvsContext::OnSueCharacterResult@0x84a04e: CInPacket::Decode1(a2) at
//	0x84a05f reads a single result byte into v2 (a2=0 immediately after, at
//	0x84a069, so no further packet reads occur). v2 then drives a five-way
//	branch (0/1/2/3/else) purely to pick a StringPool notice id
//	(2934/2935/2936/2937/2938) for a local CHATLOG_ADD render — result=2 takes
//	the v4==0 arm at 0x84a0f5 (StringPool 2936). sub_47010A@0x84a16f is
//	CInPacket/local-buffer teardown, not a wire read. Body is exactly one
//	byte, unconditionally, for every branch — byte-identical to the v83+
//	shape already coded below (only the opcode moves: 0x34 on v61 per
//	registry gms_v61.yaml op SUE_CHARACTER_RESULT vs 0x37 on v83+).
//
// packet-audit:verify packet=report/clientbound/SueCharacterResult version=gms_v61 ida=0x84a04e
func TestSueCharacterResultByteOutputV61(t *testing.T) {
	v := pt.Variants[8] // GMS v61
	if v.Name != "GMS v61" {
		t.Fatalf("pt.Variants[8] = %q, want %q (index drifted)", v.Name, "GMS v61")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	// result=2 -> StringPool 2936 branch (decompile @0x84a0f5); the wire body
	// is the raw byte regardless of value.
	input := NewSueCharacterResult(0x02)
	expected := []byte{0x02} // 1 byte, per Decode1 @0x84a05f
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestSueCharacterResultByteOutputV72 verifies the wire-exact byte output of
// SueCharacterResult for GMS v72.
// IDA evidence (session c8acae95, GMS_v72.1_U_DEVM.exe.i64):
//
//	CWvsContext::OnSueCharacterResult@0x9216f0: CInPacket::Decode1(a2) at
//	0x921701 reads a single result byte into v2 (a2=0 immediately after, at
//	0x92170b, so no further packet reads occur). v2 then drives a five-way
//	branch (0/1/2/3/else) purely to pick a StringPool notice id: 0->2979
//	@0x9217ed, 1->2980 @0x9217bf, 2->2981 @0x921791, 3->2982 @0x921760,
//	else->2983 @0x92172f — for a local CHATLOG_ADD render. sub_480FF9(&a2,12)
//	@0x92181b is local cleanup, not a wire read. Body is exactly one byte,
//	unconditionally,
//	for every branch — resolved directly via the client's opcode dispatch table
//	CWvsContext::OnPacket case 52 @0x90270f (registry gms_v72.yaml op
//	SUE_CHARACTER_RESULT, opcode 52/0x34), byte-identical to the v61 shape
//	already verified above (only the opcode moves per-version).
//
// packet-audit:verify packet=report/clientbound/SueCharacterResult version=gms_v72 ida=0x9216f0
func TestSueCharacterResultByteOutputV72(t *testing.T) {
	v := pt.Variants[9] // GMS v72
	if v.Name != "GMS v72" {
		t.Fatalf("pt.Variants[9] = %q, want %q (index drifted)", v.Name, "GMS v72")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	// result=2 -> StringPool 2981 branch (decompile @0x921791); the wire body
	// is the raw byte regardless of value.
	input := NewSueCharacterResult(0x02)
	expected := []byte{0x02} // 1 byte, per Decode1 @0x921701
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

func TestSueCharacterResultRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewSueCharacterResult(0x04)
			output := SueCharacterResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Result() != input.Result() {
				t.Errorf("round-trip mismatch: got %d want %d", output.Result(), input.Result())
			}
		})
	}
}
