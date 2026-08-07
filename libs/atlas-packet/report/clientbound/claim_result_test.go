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

// TestClaimResultSuccessByteOutputV79 verifies the wire-exact byte output of
// ClaimResultSuccess (mode 2) for GMS v79.
// IDA evidence (session 1438cecd, GMS_v79_1_DEVM.exe.i64):
//
//	CWvsContext::OnClaimResult@0x9718f4, resolved via the opcode dispatch table
//	CWvsContext::OnPacket case 42 @0x953954 (registry gms_v79.yaml op
//	CLAIM_RESULT, opcode 42/0x2A). CInPacket::Decode1(a2) @0x971909 reads the
//	mode byte (v3). mode==2 is the ONLY value that reads further: Decode1(a2)
//	@0x971988 reads hasRemaining (v8), Decode4(a2) @0x971992 reads remaining
//	(a2, reused as an int32). Every other reachable mode (3, 0x41-0x45, 0x47,
//	0x48 -- see the registry note on this op) is a bare mode byte with no
//	further packet reads -- ClaimResultNotice, verified below. 6 bytes total
//	for mode 2: mode, hasRemaining, remaining(4).
//
// packet-audit:verify packet=report/clientbound/ClaimResultSuccess version=gms_v79 ida=0x9718f4
func TestClaimResultSuccessByteOutputV79(t *testing.T) {
	v := pt.Variants[10] // GMS v79
	if v.Name != "GMS v79" {
		t.Fatalf("pt.Variants[10] = %q, want %q (index drifted)", v.Name, "GMS v79")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimResultSuccess(0x02, true, 100)
	expected := []byte{
		0x02,                   // Decode1 mode @0x971909
		0x01,                   // Decode1 hasRemaining @0x971988
		0x64, 0x00, 0x00, 0x00, // Decode4 remaining = 100 LE @0x971992
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimResultNoticeByteOutputV79 verifies the wire-exact byte output of
// ClaimResultNotice (bare mode byte, no payload) for GMS v79, covering every
// reachable non-success mode.
// IDA evidence (session 1438cecd, GMS_v79_1_DEVM.exe.i64), all within
// CWvsContext::OnClaimResult@0x9718f4 after the single Decode1(mode)
// @0x971909 -- none of these branches perform any further CInPacket read:
//
//	mode 3    -> StringPool 3364 @0x971970
//	mode 0x41 -> StringPool 3366 @0x97195a
//	mode 0x42 -> StringPool 3367 @0x971944
//	mode 0x43 -> StringPool 3368 @0x971aa2
//	mode 0x44 -> StringPool 3369 @0x971b76
//	mode 0x45 -> StringPool 3370 @0x971b62
//	mode 0x47 -> StringPool 3375 @0x971aed (own CHATLOG_ADD+Notice arm)
//	mode 0x48 -> StringPool 3377 @0x971ad3
//
// (modes 0, 1, 0x46, and every other unlisted byte value fall through to an
// early return with no display -- still just the 1-byte mode already
// consumed.) Not registry-linked (the CLAIM_RESULT op row's single `packet:`
// field points at ClaimResultSuccess above) so this test carries no
// verify-marker comment, but the decompile citations are load-bearing for
// the registry note on CLAIM_RESULT.
func TestClaimResultNoticeByteOutputV79(t *testing.T) {
	v := pt.Variants[10] // GMS v79
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

// TestClaimResultSuccessByteOutputV83 verifies the wire-exact byte output of
// ClaimResultSuccess (mode 2) for GMS v83.
// IDA evidence (session 41f13e0d, v83_Me MapleStory_dump.exe.i64):
//
//	CWvsContext::OnClaimResult@0xa27891, resolved via the opcode dispatch table
//	CWvsContext::OnPacket case 0x2D @0xa07b5c (registry gms_v83.yaml op
//	CLAIM_RESULT, opcode 45/0x2D). CInPacket::Decode1(a2) @0xa278a6 reads the
//	mode byte (v3). mode==2 is the ONLY value that reads further:
//	CInPacket::Decode1(a2) @0xa27925 reads hasRemaining (v8),
//	CInPacket::Decode4(a2) @0xa2792f reads remaining (a2, reused as an int32).
//	Every other reachable mode (3, 0x41-0x45, 0x47, 0x48) is a bare mode byte
//	with no further packet reads -- ClaimResultNotice, verified below. 6 bytes
//	total for mode 2: mode, hasRemaining, remaining(4). Byte-identical to the
//	v72/v79 shape already verified above.
//
// packet-audit:verify packet=report/clientbound/ClaimResultSuccess version=gms_v83 ida=0xa27891
func TestClaimResultSuccessByteOutputV83(t *testing.T) {
	v := pt.Variants[1] // GMS v83
	if v.Name != "GMS v83" {
		t.Fatalf("pt.Variants[1] = %q, want %q (index drifted)", v.Name, "GMS v83")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimResultSuccess(0x02, true, 100)
	expected := []byte{
		0x02,                   // Decode1 mode @0xa278a6
		0x01,                   // Decode1 hasRemaining @0xa27925
		0x64, 0x00, 0x00, 0x00, // Decode4 remaining = 100 LE @0xa2792f
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimResultNoticeByteOutputV83 verifies the wire-exact byte output of
// ClaimResultNotice (bare mode byte, no payload) for GMS v83, covering every
// reachable non-success mode.
// IDA evidence (session 41f13e0d, v83_Me MapleStory_dump.exe.i64), all within
// CWvsContext::OnClaimResult@0xa27891 after the single Decode1(mode)
// @0xa278a6 -- none of these branches perform any further CInPacket read:
//
//	mode 3    -> v24=SP_3384 (v3-2==0 branch) @0xa27908
//	mode 0x41 -> v24=SP_3386 (v7==0 branch) @0xa278f2
//	mode 0x42 -> v24=SP_3387 (v7==1 branch) @0xa278dc
//	mode 0x43 -> v24=SP_3388 (v3==67 branch) @0xa27a3a
//	mode 0x44 -> v24=SP_3389 (v17==0 branch) @0xa27b0d
//	mode 0x45 -> v24=SP_3390 (v17==1 branch) @0xa27afa
//	mode 0x47 -> own CUtilDlg::Notice arm (v18==2 branch, SP_3395 formatted
//	             with the claim-count range) @0xa27a8a-0xa27ad8
//	mode 0x48 -> v24=SP_3397 (v19==1 branch) @0xa27a6b
//
// (modes 0, 1, 0x46, and every other unlisted byte value fall through to an
// early return with no display -- still just the 1-byte mode already
// consumed.) Not registry-linked (the CLAIM_RESULT op row's single `packet:`
// field points at ClaimResultSuccess above) so this test carries no
// verify-marker comment, but the decompile citations are load-bearing for
// the registry note on CLAIM_RESULT.
func TestClaimResultNoticeByteOutputV83(t *testing.T) {
	v := pt.Variants[1] // GMS v83
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

// TestClaimResultSuccessByteOutputV84 verifies the wire-exact byte output of
// ClaimResultSuccess (mode 2) for GMS v84.
// IDA evidence (session 5881cf84, GMS_v84.1_U_DEVM.i64):
//
//	CWvsContext::OnClaimResult@0xa7304c (named this pass; was sub_A7304C),
//	resolved via the opcode dispatch table CWvsContext::OnPacket@0xa51cd0
//	case 0x2D, call-site @0xa51e31 (registry gms_v84.yaml op CLAIM_RESULT,
//	opcode 45/0x2D -- confirmed identical to v83, NOT shifted despite the
//	documented post-0x3D v84 shift class; this op's dispatch entry sits
//	below 0x3D). CInPacket::Decode1(a2) @0xa73061 reads the mode byte (v3).
//	mode==2 is the ONLY value that reads further: CInPacket::Decode1(a2)
//	@0xa730e0 reads hasRemaining (v8), CInPacket::Decode4(a2) @0xa730ea reads
//	remaining (a2, reused as an int32). Every other reachable mode (3,
//	0x41-0x45, 0x47, 0x48) is a bare mode byte with no further packet reads
//	-- ClaimResultNotice, verified below. 6 bytes total for mode 2: mode,
//	hasRemaining, remaining(4). Byte-identical to the v72/v79/v83 shape
//	already verified above.
//
// packet-audit:verify packet=report/clientbound/ClaimResultSuccess version=gms_v84 ida=0xa7304c
func TestClaimResultSuccessByteOutputV84(t *testing.T) {
	v := pt.Variants[5] // GMS v84
	if v.Name != "GMS v84" {
		t.Fatalf("pt.Variants[5] = %q, want %q (index drifted)", v.Name, "GMS v84")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimResultSuccess(0x02, true, 100)
	expected := []byte{
		0x02,                   // Decode1 mode @0xa73061
		0x01,                   // Decode1 hasRemaining @0xa730e0
		0x64, 0x00, 0x00, 0x00, // Decode4 remaining = 100 LE @0xa730ea
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimResultNoticeByteOutputV84 verifies the wire-exact byte output of
// ClaimResultNotice (bare mode byte, no payload) for GMS v84, covering every
// reachable non-success mode.
// IDA evidence (session 5881cf84, GMS_v84.1_U_DEVM.i64), all within
// CWvsContext::OnClaimResult@0xa7304c after the single Decode1(mode)
// @0xa73061 -- none of these branches perform any further CInPacket read:
//
//	mode 3    -> StringPool 3387 @0xa730c3
//	mode 0x41 -> StringPool 3389 @0xa730ad
//	mode 0x42 -> StringPool 3390 @0xa73097
//	mode 0x43 -> StringPool 3391 @0xa731f5
//	mode 0x44 -> StringPool 3392 @0xa732c8
//	mode 0x45 -> StringPool 3393 @0xa732b5
//	mode 0x47 -> own arm reading this[12556]/this[12557] (the openHour/
//	             closeHour fields stored by OnSetClaimSvrAvailableTime),
//	             StringPool 3398 @0xa73232-0xa73258 -- reads character state,
//	             not the wire, so still zero further CInPacket reads
//	mode 0x48 -> StringPool 3400 @0xa73226
//
// (modes 0, 1, 0x46, and every other unlisted byte value fall through to an
// early return with no display -- still just the 1-byte mode already
// consumed.) Not registry-linked (the CLAIM_RESULT op row's single `packet:`
// field points at ClaimResultSuccess above) so this test carries no
// verify-marker comment, but the decompile citations are load-bearing for
// the registry note on CLAIM_RESULT.
func TestClaimResultNoticeByteOutputV84(t *testing.T) {
	v := pt.Variants[5] // GMS v84
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

// TestClaimResultSuccessByteOutputV87 verifies the wire-exact byte output of
// ClaimResultSuccess (mode 2) for GMS v87.
// IDA evidence (session d51ecbd3, GMSv87_4GB.exe.i64):
//
//	CWvsContext::OnClaimResult@0xabf4fe, resolved via the opcode dispatch
//	table CWvsContext::OnPacket@0xa9d011 case 0x2D @0xa9d172 (registry
//	gms_v87.yaml op CLAIM_RESULT, opcode 45/0x2D -- STATUS.md's pre-filled
//	v87 column value of 0x02E is stale/wrong; independently re-derived here
//	from the live dispatch switch, which agrees with the registry).
//	CInPacket::Decode1(a2) @0xabf513 reads the mode byte (v3). mode==2 is
//	the ONLY value that reads further: CInPacket::Decode1(a2) @0xabf592
//	reads hasRemaining (v8), CInPacket::Decode4(a2) @0xabf59c reads
//	remaining (a2, reused as an int32). Every other reachable mode (3,
//	0x41-0x45, 0x47, 0x48) is a bare mode byte with no further packet reads
//	-- ClaimResultNotice, verified below. 6 bytes total for mode 2: mode,
//	hasRemaining, remaining(4). Byte-identical to the v72/v79/v83/v84 shape
//	already verified above.
//
// packet-audit:verify packet=report/clientbound/ClaimResultSuccess version=gms_v87 ida=0xabf4fe
func TestClaimResultSuccessByteOutputV87(t *testing.T) {
	v := pt.Variants[2] // GMS v87
	if v.Name != "GMS v87" {
		t.Fatalf("pt.Variants[2] = %q, want %q (index drifted)", v.Name, "GMS v87")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimResultSuccess(0x02, true, 100)
	expected := []byte{
		0x02,                   // Decode1 mode @0xabf513
		0x01,                   // Decode1 hasRemaining @0xabf592
		0x64, 0x00, 0x00, 0x00, // Decode4 remaining = 100 LE @0xabf59c
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimResultNoticeByteOutputV87 verifies the wire-exact byte output of
// ClaimResultNotice (bare mode byte, no payload) for GMS v87, covering every
// reachable non-success mode.
// IDA evidence (session d51ecbd3, GMSv87_4GB.exe.i64), all within
// CWvsContext::OnClaimResult@0xabf4fe after the single Decode1(mode)
// @0xabf513 -- none of these branches perform any further CInPacket read:
//
//	mode 3    -> StringPool 3394 @0xabf575
//	mode 0x41 -> StringPool 5875 @0xabf55f
//	mode 0x42 -> StringPool 3396 @0xabf549
//	mode 0x43 -> StringPool 3397 @0xabf6a7
//	mode 0x44 -> StringPool 3398 @0xabf77a
//	mode 0x45 -> StringPool 3399 @0xabf767
//	mode 0x47 -> own CUtilDlg::Notice arm (formats StringPool 3404 with
//	             this[12649]/this[12648], the openHour/closeHour fields
//	             stored by OnSetClaimSvrAvailableTime -- reads character
//	             state, not the wire) @0xabf6f7-0xabf745
//	mode 0x48 -> StringPool 3406 @0xabf6d8
//
// (modes 0, 1, 0x46, and every other unlisted byte value fall through to an
// early return with no display -- still just the 1-byte mode already
// consumed.) Not registry-linked (the CLAIM_RESULT op row's single `packet:`
// field points at ClaimResultSuccess above) so this test carries no
// verify-marker comment, but the decompile citations are load-bearing for
// the registry note on CLAIM_RESULT.
func TestClaimResultNoticeByteOutputV87(t *testing.T) {
	v := pt.Variants[2] // GMS v87
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

// TestClaimResultSuccessByteOutputV92 verifies the wire-exact byte output of
// ClaimResultSuccess (mode 2) for GMS v92.
// IDA evidence (session acdfccff, GMS_v92_1_DEVM.exe.i64):
//
//	CWvsContext::OnClaimResult@0x9cf310, resolved via the opcode dispatch
//	table CWvsContext::OnPacket@0x9ba740 case 46 @0x9ba8ae (registry
//	gms_v92.yaml op CLAIM_RESULT, opcode 46/0x2E -- matches STATUS.md's
//	pre-filled v92 column value of 0x02E, and matches the live dispatch
//	switch, independently re-derived here). CInPacket::Decode1(a2)
//	@0x9cf346 reads the raw mode byte; the function immediately subtracts
//	2 and switches on that. mode==2 (switch key 0) is the ONLY value that
//	reads further: CInPacket::Decode1(a2) @0x9cf379 reads hasRemaining
//	(v6), CInPacket::Decode4(a2) @0x9cf383 reads remaining (v7). Every
//	other reachable mode (3, 0x41-0x45, 0x47, 0x48) is a bare mode byte
//	with no further packet reads -- ClaimResultNotice, verified below.
//	6 bytes total for mode 2: mode, hasRemaining, remaining(4).
//	Byte-identical to the v72/v79/v83/v84/v87 shape already verified above.
//
// packet-audit:verify packet=report/clientbound/ClaimResultSuccess version=gms_v92 ida=0x9cf310
func TestClaimResultSuccessByteOutputV92(t *testing.T) {
	v := pt.Variants[11] // GMS v92
	if v.Name != "GMS v92" {
		t.Fatalf("pt.Variants[11] = %q, want %q (index drifted)", v.Name, "GMS v92")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimResultSuccess(0x02, true, 100)
	expected := []byte{
		0x02,                   // Decode1 mode @0x9cf346
		0x01,                   // Decode1 hasRemaining @0x9cf379
		0x64, 0x00, 0x00, 0x00, // Decode4 remaining = 100 LE @0x9cf383
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimResultNoticeByteOutputV92 verifies the wire-exact byte output of
// ClaimResultNotice (bare mode byte, no payload) for GMS v92, covering every
// reachable non-success mode.
// IDA evidence (session acdfccff, GMS_v92_1_DEVM.exe.i64), all within
// CWvsContext::OnClaimResult@0x9cf310 after the single Decode1(mode)
// @0x9cf346 (raw byte, switch key = raw-2) -- none of these branches
// perform any further CInPacket read:
//
//	mode 3    -> switch key 1,    StringPool 3449 @0x9cf499
//	mode 0x41 -> switch key 0x3F, StringPool 6309 @0x9cf4d7
//	mode 0x42 -> switch key 0x40, StringPool 3451 @0x9cf515
//	mode 0x43 -> switch key 0x41, StringPool 3452 @0x9cf552
//	mode 0x44 -> switch key 0x42, StringPool 3453 @0x9cf56c
//	mode 0x45 -> switch key 0x43, StringPool 3454 @0x9cf586
//	mode 0x47 -> switch key 0x45, StringPool 3459 @0x9cf5bf -- own arm
//	             reading this+13892/this+13893 (the openHour/closeHour
//	             fields stored by OnSetClaimSvrAvailableTime) -- reads
//	             character state, not the wire, so still zero further
//	             CInPacket reads
//	mode 0x48 -> switch key 0x46, StringPool 3461 @0x9cf59f
//
// (modes 0, 1, 0x46, and every other unlisted byte value map to a switch
// key with no case -- default: silent return, still just the 1-byte mode
// already consumed.) Not registry-linked (the CLAIM_RESULT op row's single
// `packet:` field points at ClaimResultSuccess above) so this test carries
// no verify-marker comment, but the decompile citations are load-bearing
// for the registry note on CLAIM_RESULT.
func TestClaimResultNoticeByteOutputV92(t *testing.T) {
	v := pt.Variants[11] // GMS v92
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

// TestClaimResultSuccessByteOutputV95 verifies the wire-exact byte output of
// ClaimResultSuccess (mode 2) for GMS v95.
// IDA evidence (session 79906a1e, GMS_v95.0_U_DEVM.exe.i64):
//
//	CWvsContext::OnClaimResult@0x9fa7d0, resolved via the opcode dispatch
//	table CWvsContext::OnPacket@0x9e5830 case 44, call-site @0x9e599e
//	(registry gms_v95.yaml op CLAIM_RESULT, opcode 44/0x2C -- matches
//	STATUS.md's pre-filled v95 column value of 0x02C). CInPacket::Decode1
//	(iPacket) @0x9fa819 (inlined into the switch) reads the mode byte.
//	mode==2 is the ONLY value that reads further: CInPacket::Decode1(v3)
//	@0x9fa839 reads hasRemaining (v5), CInPacket::Decode4(v3) @0x9fa843
//	reads remaining (v6). Every other reachable mode (3, 0x41-0x45, 0x47,
//	0x48) is a bare mode byte with no further packet reads --
//	ClaimResultNotice, verified below. 6 bytes total for mode 2: mode,
//	hasRemaining, remaining(4). Byte-identical to the
//	v72/v79/v83/v84/v87/v92 shape already verified above.
//
// packet-audit:verify packet=report/clientbound/ClaimResultSuccess version=gms_v95 ida=0x9fa7d0
func TestClaimResultSuccessByteOutputV95(t *testing.T) {
	v := pt.Variants[3] // GMS v95
	if v.Name != "GMS v95" {
		t.Fatalf("pt.Variants[3] = %q, want %q (index drifted)", v.Name, "GMS v95")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimResultSuccess(0x02, true, 100)
	expected := []byte{
		0x02,                   // Decode1 mode (inlined in switch) @0x9fa819
		0x01,                   // Decode1 hasRemaining @0x9fa839
		0x64, 0x00, 0x00, 0x00, // Decode4 remaining = 100 LE @0x9fa843
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimResultNoticeByteOutputV95 verifies the wire-exact byte output of
// ClaimResultNotice (bare mode byte, no payload) for GMS v95, covering every
// reachable non-success mode.
// IDA evidence (session 79906a1e, GMS_v95.0_U_DEVM.exe.i64), all within
// CWvsContext::OnClaimResult@0x9fa7d0 after the single Decode1(mode)
// @0x9fa819 -- none of these branches perform any further CInPacket read:
//
//	mode 3    -> StringPool 3417 (v20=3417) @0x9fa953, via LABEL_13 @0x9fa959
//	mode 0x41 -> StringPool 0x1A5E @0x9fa99e -- own CUtilDlg::Notice arm
//	mode 0x42 -> StringPool 0xD5B @0x9fa9dc -- own CUtilDlg::Notice arm
//	mode 0x43 -> StringPool 3420 (v20=3420) @0x9faa0d, via LABEL_13
//	mode 0x44 -> StringPool 3421 (v20=3421) @0x9faa26, via LABEL_13
//	mode 0x45 -> StringPool 3422 (v20=3422) @0x9faa40, via LABEL_13
//	mode 0x47 -> StringPool 0xD63 @0x9faa86, formatted with
//	             this->m_nClaimSvrOpenTime/m_nClaimSvrCloseTime (character
//	             state, not the wire) -- own arm via LABEL_22 @0x9faac8
//	mode 0x48 -> StringPool 3429 (v20=3429) @0x9faa5a, via LABEL_13
//
// (modes 0, 1, 0x46, and every other unlisted byte value fall through to
// `default: return` -- still just the 1-byte mode already consumed.) Not
// registry-linked (the CLAIM_RESULT op row's single `packet:` field points
// at ClaimResultSuccess above) so this test carries no verify-marker
// comment, but the decompile citations are load-bearing for the registry
// note on CLAIM_RESULT.
func TestClaimResultNoticeByteOutputV95(t *testing.T) {
	v := pt.Variants[3] // GMS v95
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
