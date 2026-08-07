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

// TestSueCharacterResultByteOutputV79 verifies the wire-exact byte output of
// SueCharacterResult for GMS v79.
// IDA evidence (session 1438cecd, GMS_v79_1_DEVM.exe.i64):
//
//	CWvsContext::OnSueCharacterResult@0x9735a5, resolved via the opcode
//	dispatch table CWvsContext::OnPacket case 52 @0x953947 (registry
//	gms_v79.yaml op SUE_CHARACTER_RESULT, opcode 52/0x34). CInPacket::Decode1(a2)
//	@0x9735b6 reads a single result byte (v2). v2 then drives a five-way
//	branch (0/1/2/3/else) purely to pick a StringPool notice id
//	(2982/2983/2984/2985/2986 @0x9736a2/0x973674/0x973646/0x973615/0x9735e4)
//	for a local CHATLOG_ADD render -- no branch performs any further
//	CInPacket read. sub_4888C3(&a2,12) @0x9736d0 is local teardown, not a
//	wire read. Body is exactly one byte, unconditionally, for every branch --
//	byte-identical to the v61/v72 shape already verified above (only the
//	opcode moves per-version).
//
// packet-audit:verify packet=report/clientbound/SueCharacterResult version=gms_v79 ida=0x9735a5
func TestSueCharacterResultByteOutputV79(t *testing.T) {
	v := pt.Variants[10] // GMS v79
	if v.Name != "GMS v79" {
		t.Fatalf("pt.Variants[10] = %q, want %q (index drifted)", v.Name, "GMS v79")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	// result=2 -> StringPool 2984 branch (decompile @0x973646); the wire body
	// is the raw byte regardless of value.
	input := NewSueCharacterResult(0x02)
	expected := []byte{0x02} // 1 byte, per Decode1 @0x9735b6
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestSueCharacterResultByteOutputV83 verifies the wire-exact byte output of
// SueCharacterResult for GMS v83.
// IDA evidence (session 41f13e0d, v83_Me MapleStory_dump.exe.i64):
//
//	CWvsContext::OnSueCharacterResult@0xa29739, resolved via the opcode
//	dispatch table CWvsContext::OnPacket case 0x37 @0xa07b4f (registry
//	gms_v83.yaml op SUE_CHARACTER_RESULT, opcode 55/0x37). CInPacket::Decode1(a2)
//	@0xa2974a reads a single result byte (v2). v2 then drives a five-way
//	branch (0/1/2/3/else) purely to pick a StringPool notice id
//	(SP_3003/3004/3005/3006/3007) for a local CHATLOG_ADD render -- no branch
//	performs any further CInPacket read. Body is exactly one byte,
//	unconditionally, for every branch -- byte-identical to the v61/v72/v79
//	shape already verified above (only the opcode moves per-version: 0x37 on
//	v83+ vs 0x34 on v61/v72/v79).
//
// packet-audit:verify packet=report/clientbound/SueCharacterResult version=gms_v83 ida=0xa29739
func TestSueCharacterResultByteOutputV83(t *testing.T) {
	v := pt.Variants[1] // GMS v83
	if v.Name != "GMS v83" {
		t.Fatalf("pt.Variants[1] = %q, want %q (index drifted)", v.Name, "GMS v83")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	// result=2 -> StringPool SP_3005 branch (decompile @0xa297da); the wire body
	// is the raw byte regardless of value.
	input := NewSueCharacterResult(0x02)
	expected := []byte{0x02} // 1 byte, per Decode1 @0xa2974a
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestSueCharacterResultByteOutputV84 verifies the wire-exact byte output of
// SueCharacterResult for GMS v84.
// IDA evidence (session 5881cf84, GMS_v84.1_U_DEVM.i64):
//
//	CWvsContext::OnSueCharacterResult@0xa74efc (named this pass; was
//	sub_A74EFC), resolved via the opcode dispatch table
//	CWvsContext::OnPacket@0xa51cd0 case 0x37, call-site @0xa51e24 (registry
//	gms_v84.yaml op SUE_CHARACTER_RESULT, opcode 55/0x37 -- confirmed
//	identical to v83, not shifted despite this op's opcode sitting past the
//	documented post-0x3D shift boundary; the v84 dispatch table itself is
//	NOT shifted for this range, only specific ops in other families are).
//	CInPacket::Decode1(a1) @0xa74f0d reads a single result byte (v1). v1
//	then drives a five-way branch (0/1/2/3/else) purely to pick a StringPool
//	notice id (3006/3007/3008/3009/3010 @0xa75005/0xa74fd7/0xa74fa9/
//	0xa74f78/0xa74f47) for a local render -- no branch performs any further
//	CInPacket read. Body is exactly one byte, unconditionally, for every
//	branch -- byte-identical to the v61/v72/v79/v83 shape already verified
//	above (only the opcode moves per-version: 0x37 on v83+ vs 0x34 on
//	v61/v72/v79).
//
// packet-audit:verify packet=report/clientbound/SueCharacterResult version=gms_v84 ida=0xa74efc
func TestSueCharacterResultByteOutputV84(t *testing.T) {
	v := pt.Variants[5] // GMS v84
	if v.Name != "GMS v84" {
		t.Fatalf("pt.Variants[5] = %q, want %q (index drifted)", v.Name, "GMS v84")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	// result=2 -> StringPool 3008 branch (decompile @0xa74fa9); the wire body
	// is the raw byte regardless of value.
	input := NewSueCharacterResult(0x02)
	expected := []byte{0x02} // 1 byte, per Decode1 @0xa74f0d
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestSueCharacterResultByteOutputV87 verifies the wire-exact byte output of
// SueCharacterResult for GMS v87.
// IDA evidence (session d51ecbd3, GMSv87_4GB.exe.i64):
//
//	CWvsContext::OnSueCharacterResult@0xac13af, resolved via the opcode
//	dispatch table CWvsContext::OnPacket@0xa9d011 case 0x37 @0xa9d165
//	(registry gms_v87.yaml op SUE_CHARACTER_RESULT, opcode 55/0x37 --
//	STATUS.md's pre-filled v87 column value of 0x038 is stale/wrong;
//	independently re-derived here from the live dispatch switch, which
//	agrees with the registry). CInPacket::Decode1(a1) @0xac13c0 reads a
//	single result byte (v1); a1 is zeroed immediately after (@0xac13ca), so
//	no further packet reads occur. v1 then drives a five-way branch
//	(0/1/2/3/else) purely to pick a StringPool notice id
//	(3013/3014/3015/3016/3017 @0xac14ac/0xac147e/0xac1450/0xac141f/0xac13ee)
//	for a local CHATLOG_ADD render (@0xac14da) -- no branch performs any
//	further CInPacket read. Body is exactly one byte, unconditionally, for
//	every branch -- byte-identical to the v61/v72/v79/v83/v84 shape already
//	verified above.
//
// packet-audit:verify packet=report/clientbound/SueCharacterResult version=gms_v87 ida=0xac13af
func TestSueCharacterResultByteOutputV87(t *testing.T) {
	v := pt.Variants[2] // GMS v87
	if v.Name != "GMS v87" {
		t.Fatalf("pt.Variants[2] = %q, want %q (index drifted)", v.Name, "GMS v87")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	// result=2 -> StringPool 3015 branch (decompile @0xac1450); the wire body
	// is the raw byte regardless of value.
	input := NewSueCharacterResult(0x02)
	expected := []byte{0x02} // 1 byte, per Decode1 @0xac13c0
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestSueCharacterResultByteOutputV92 verifies the wire-exact byte output of
// SueCharacterResult for GMS v92.
// IDA evidence (session acdfccff, GMS_v92_1_DEVM.exe.i64):
//
//	CWvsContext::OnSueCharacterResult@0x9cf950, resolved via the opcode
//	dispatch table CWvsContext::OnPacket@0x9ba740 case 56 @0x9ba8a1
//	(registry gms_v92.yaml op SUE_CHARACTER_RESULT, opcode 56/0x38 --
//	matches STATUS.md's pre-filled v92 column value of 0x038, and matches
//	the live dispatch switch, independently re-derived here).
//	CInPacket::Decode1(a2) @0x9cf979 reads a single result byte (v2); a2 is
//	zeroed immediately after (@0x9cf983), so no further packet reads occur.
//	v2 then drives a five-way branch (0/1/2/3/else) purely to pick a
//	StringPool notice id (3065/3066/3067/3068/3069 @0x9cf9a5/0x9cf9d6/
//	0x9cfa07/0x9cfa35/0x9cfa63) -- no branch performs any further
//	CInPacket read. Body is exactly one byte, unconditionally, for every
//	branch -- byte-identical to the v61/v72/v79/v83/v84/v87 shape already
//	verified above.
//
// packet-audit:verify packet=report/clientbound/SueCharacterResult version=gms_v92 ida=0x9cf950
func TestSueCharacterResultByteOutputV92(t *testing.T) {
	v := pt.Variants[11] // GMS v92
	if v.Name != "GMS v92" {
		t.Fatalf("pt.Variants[11] = %q, want %q (index drifted)", v.Name, "GMS v92")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	// result=2 -> StringPool 3067 branch (decompile @0x9cfa07); the wire body
	// is the raw byte regardless of value.
	input := NewSueCharacterResult(0x02)
	expected := []byte{0x02} // 1 byte, per Decode1 @0x9cf979
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestSueCharacterResultByteOutputV95 verifies the wire-exact byte output of
// SueCharacterResult for GMS v95.
// IDA evidence (session 79906a1e, GMS_v95.0_U_DEVM.exe.i64):
//
//	CWvsContext::OnSueCharacterResult@0x9fae10, resolved via the opcode
//	dispatch table CWvsContext::OnPacket@0x9e5830 case 55, call-site
//	@0x9e5991 (registry gms_v95.yaml op SUE_CHARACTER_RESULT, opcode
//	55/0x37 -- matches STATUS.md's pre-filled v95 column value of 0x037).
//	CInPacket::Decode1(iPacket) @0x9fae39 reads a single result byte (v2);
//	iPacket is zeroed immediately after (@0x9fae43), so no further packet
//	reads occur. v2 then drives a five-way branch (0/1/2/3/else) purely to
//	pick a StringPool notice id (0xBE6/0xBE7/0xBE8/0xBE9/0xBEA @0x9fae73/
//	0x9faea4/0x9faed5/0x9faf03/0x9faf31) for a CUIStatusBar::ChatLogAdd
//	render (@0x9faf80) -- no branch performs any further CInPacket read.
//	Body is exactly one byte, unconditionally, for every branch --
//	byte-identical to the v61/v72/v79/v83/v84/v87/v92 shape already
//	verified above.
//
// packet-audit:verify packet=report/clientbound/SueCharacterResult version=gms_v95 ida=0x9fae10
func TestSueCharacterResultByteOutputV95(t *testing.T) {
	v := pt.Variants[3] // GMS v95
	if v.Name != "GMS v95" {
		t.Fatalf("pt.Variants[3] = %q, want %q (index drifted)", v.Name, "GMS v95")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	// result=2 -> StringPool 0xBE8 branch (decompile @0x9faed5); the wire body
	// is the raw byte regardless of value.
	input := NewSueCharacterResult(0x02)
	expected := []byte{0x02} // 1 byte, per Decode1 @0x9fae39
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
