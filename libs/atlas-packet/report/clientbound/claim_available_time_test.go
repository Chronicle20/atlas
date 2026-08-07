package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestClaimAvailableTimeGolden(t *testing.T) {
	// openHour, closeHour. 0/0 = always available (verified client branch).
	input := NewClaimAvailableTime(8, 22)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x08, 0x16}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimAvailableTimeAlwaysOpenGolden(t *testing.T) {
	input := NewClaimAvailableTime(0, 0)
	ctx := pt.CreateContext("GMS", 95, 1)
	expected := []byte{0x00, 0x00}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimAvailableTimeByteOutputV72 verifies the wire-exact byte output of
// ClaimAvailableTime for GMS v72.
// IDA evidence (session c8acae95, GMS_v72.1_U_DEVM.exe.i64):
//
//	CWvsContext::OnSetClaimSvrAvailableTime@0x91fc50, resolved via the opcode
//	dispatch table CWvsContext::OnPacket case 43 @0x902729 (registry
//	gms_v72.yaml op CLAIM_AVAILABLE_TIME, opcode 43/0x2B). CInPacket::Decode1(a2)
//	@0x91fc61 -> openHour (v3, stored at this+11888). CInPacket::Decode1(a2)
//	@0x91fc63 -> closeHour (result, stored at this+11889). Function body is
//	exactly these two Decode1 calls plus the two field stores and a return —
//	no branches. 2 bytes total.
//
// packet-audit:verify packet=report/clientbound/ClaimAvailableTime version=gms_v72 ida=0x91fc50
func TestClaimAvailableTimeByteOutputV72(t *testing.T) {
	v := pt.Variants[9] // GMS v72
	if v.Name != "GMS v72" {
		t.Fatalf("pt.Variants[9] = %q, want %q (index drifted)", v.Name, "GMS v72")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimAvailableTime(8, 22)
	expected := []byte{0x08, 0x16} // Decode1 openHour @0x91fc61, Decode1 closeHour @0x91fc63
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimAvailableTimeByteOutputV79 verifies the wire-exact byte output of
// ClaimAvailableTime for GMS v79.
// IDA evidence (session 1438cecd, GMS_v79_1_DEVM.exe.i64):
//
//	CWvsContext::OnSetClaimSvrAvailableTime@0x971b9b, resolved via the opcode
//	dispatch table CWvsContext::OnPacket case 43 @0x953961 (registry
//	gms_v79.yaml op CLAIM_AVAILABLE_TIME, opcode 43/0x2B). CInPacket::Decode1(a2)
//	@0x971bac -> openHour (v3, stored at this+12116). CInPacket::Decode1(a2)
//	@0x971bae -> closeHour (result, stored at this+12117). Function body is
//	exactly these two Decode1 calls plus the two field stores and a return —
//	no branches. 2 bytes total.
//
// packet-audit:verify packet=report/clientbound/ClaimAvailableTime version=gms_v79 ida=0x971b9b
func TestClaimAvailableTimeByteOutputV79(t *testing.T) {
	v := pt.Variants[10] // GMS v79
	if v.Name != "GMS v79" {
		t.Fatalf("pt.Variants[10] = %q, want %q (index drifted)", v.Name, "GMS v79")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimAvailableTime(8, 22)
	expected := []byte{0x08, 0x16} // Decode1 openHour @0x971bac, Decode1 closeHour @0x971bae
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimAvailableTimeByteOutputV83 verifies the wire-exact byte output of
// ClaimAvailableTime for GMS v83.
// IDA evidence (session 41f13e0d, v83_Me MapleStory_dump.exe.i64):
//
//	CWvsContext::OnSetClaimSvrAvailableTime@0xa27b38, resolved via the opcode
//	dispatch table CWvsContext::OnPacket case 0x2E @0xa07b69 (registry
//	gms_v83.yaml op CLAIM_AVAILABLE_TIME, opcode 46/0x2E). CInPacket::Decode1(a2)
//	@0xa27b49 -> openHour (v3, stored at this[12476]). CInPacket::Decode1(a2)
//	@0xa27b4b -> closeHour (v4, stored at this[12477]). Function body is
//	exactly these two Decode1 calls plus the two field stores and a return —
//	no branches. 2 bytes total. Byte-identical to the v72/v79 twins.
//
// packet-audit:verify packet=report/clientbound/ClaimAvailableTime version=gms_v83 ida=0xa27b38
func TestClaimAvailableTimeByteOutputV83(t *testing.T) {
	v := pt.Variants[1] // GMS v83
	if v.Name != "GMS v83" {
		t.Fatalf("pt.Variants[1] = %q, want %q (index drifted)", v.Name, "GMS v83")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimAvailableTime(8, 22)
	expected := []byte{0x08, 0x16} // Decode1 openHour @0xa27b49, Decode1 closeHour @0xa27b4b
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimAvailableTimeByteOutputV84 verifies the wire-exact byte output of
// ClaimAvailableTime for GMS v84.
// IDA evidence (session 5881cf84, GMS_v84.1_U_DEVM.i64):
//
//	CWvsContext::OnSetClaimSvrAvailableTime@0xa732f3 (named this pass; was
//	sub_A732F3), resolved via the opcode dispatch table
//	CWvsContext::OnPacket@0xa51cd0 case 0x2E, call-site @0xa51e3e (registry
//	gms_v84.yaml op CLAIM_AVAILABLE_TIME, opcode 46/0x2E -- confirmed
//	identical to v83, not shifted). CInPacket::Decode1(a2) @0xa73304 ->
//	openHour (v3, stored this[12556]). CInPacket::Decode1(a2) @0xa73306 ->
//	closeHour (result, stored this[12557]). Function body is exactly these
//	two Decode1 calls plus the two field stores and a return -- no branches.
//	2 bytes total. Byte-identical to the v72/v79/v83 twins.
//
// packet-audit:verify packet=report/clientbound/ClaimAvailableTime version=gms_v84 ida=0xa732f3
func TestClaimAvailableTimeByteOutputV84(t *testing.T) {
	v := pt.Variants[5] // GMS v84
	if v.Name != "GMS v84" {
		t.Fatalf("pt.Variants[5] = %q, want %q (index drifted)", v.Name, "GMS v84")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimAvailableTime(8, 22)
	expected := []byte{0x08, 0x16} // Decode1 openHour @0xa73304, Decode1 closeHour @0xa73306
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimAvailableTimeByteOutputV87 verifies the wire-exact byte output of
// ClaimAvailableTime for GMS v87.
// IDA evidence (session d51ecbd3, GMSv87_4GB.exe.i64):
//
//	CWvsContext::OnSetClaimSvrAvailableTime@0xabf7a5, resolved via the
//	opcode dispatch table CWvsContext::OnPacket@0xa9d011 case 0x2E
//	@0xa9d17f (registry gms_v87.yaml op CLAIM_AVAILABLE_TIME, opcode
//	46/0x2E -- STATUS.md's pre-filled v87 column value of 0x02F is
//	stale/wrong; independently re-derived here from the live dispatch
//	switch, which agrees with the registry). CInPacket::Decode1(a2)
//	@0xabf7b6 -> openHour (v3, stored this[12648]). CInPacket::Decode1(a2)
//	@0xabf7b8 -> closeHour (v4, stored this[12649]). Function body is
//	exactly these two Decode1 calls plus the two field stores and a
//	return -- no branches. 2 bytes total. Byte-identical to the
//	v72/v79/v83/v84 twins.
//
// packet-audit:verify packet=report/clientbound/ClaimAvailableTime version=gms_v87 ida=0xabf7a5
func TestClaimAvailableTimeByteOutputV87(t *testing.T) {
	v := pt.Variants[2] // GMS v87
	if v.Name != "GMS v87" {
		t.Fatalf("pt.Variants[2] = %q, want %q (index drifted)", v.Name, "GMS v87")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimAvailableTime(8, 22)
	expected := []byte{0x08, 0x16} // Decode1 openHour @0xabf7b6, Decode1 closeHour @0xabf7b8
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimAvailableTimeByteOutputV92 verifies the wire-exact byte output of
// ClaimAvailableTime for GMS v92.
// IDA evidence (session acdfccff, GMS_v92_1_DEVM.exe.i64):
//
//	CWvsContext::OnSetClaimSvrAvailableTime@0x9c5d30, resolved via the
//	opcode dispatch table CWvsContext::OnPacket@0x9ba740 case 47
//	@0x9ba8bb (registry gms_v92.yaml op CLAIM_AVAILABLE_TIME, opcode
//	47/0x2F -- matches STATUS.md's pre-filled v92 column value of 0x02F,
//	and matches the live dispatch switch, independently re-derived here).
//	CInPacket::Decode1(a2) @0x9c5d42 -> openHour (v3, stored this+13892).
//	CInPacket::Decode1(a2) @0x9c5d44 -> closeHour (result, stored
//	this+13893). Function body is exactly these two Decode1 calls plus
//	the two field stores and a return -- no branches. 2 bytes total.
//	Byte-identical to the v72/v79/v83/v84/v87 twins.
//
// packet-audit:verify packet=report/clientbound/ClaimAvailableTime version=gms_v92 ida=0x9c5d30
func TestClaimAvailableTimeByteOutputV92(t *testing.T) {
	v := pt.Variants[11] // GMS v92
	if v.Name != "GMS v92" {
		t.Fatalf("pt.Variants[11] = %q, want %q (index drifted)", v.Name, "GMS v92")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimAvailableTime(8, 22)
	expected := []byte{0x08, 0x16} // Decode1 openHour @0x9c5d42, Decode1 closeHour @0x9c5d44
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimAvailableTimeByteOutputV95 verifies the wire-exact byte output of
// ClaimAvailableTime for GMS v95.
// IDA evidence (session 79906a1e, GMS_v95.0_U_DEVM.exe.i64):
//
//	CWvsContext::OnSetClaimSvrAvailableTime@0x9f1620, resolved via the
//	opcode dispatch table CWvsContext::OnPacket@0x9e5830 case 45,
//	call-site @0x9e59ab (registry gms_v95.yaml op CLAIM_AVAILABLE_TIME,
//	opcode 45/0x2D -- matches STATUS.md's pre-filled v95 column value of
//	0x02D). CInPacket::Decode1(iPacket) @0x9f1632 -> openHour (v3, stored
//	this->m_nClaimSvrOpenTime). CInPacket::Decode1(iPacket) @0x9f1634 ->
//	closeHour (v4, stored this->m_nClaimSvrCloseTime). Function body is
//	exactly these two Decode1 calls plus the two field stores and a
//	return -- no branches. 2 bytes total. Byte-identical to the
//	v72/v79/v83/v84/v87/v92 twins.
//
// packet-audit:verify packet=report/clientbound/ClaimAvailableTime version=gms_v95 ida=0x9f1620
func TestClaimAvailableTimeByteOutputV95(t *testing.T) {
	v := pt.Variants[3] // GMS v95
	if v.Name != "GMS v95" {
		t.Fatalf("pt.Variants[3] = %q, want %q (index drifted)", v.Name, "GMS v95")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimAvailableTime(8, 22)
	expected := []byte{0x08, 0x16} // Decode1 openHour @0x9f1632, Decode1 closeHour @0x9f1634
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimAvailableTimeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClaimAvailableTime(9, 21)
			output := ClaimAvailableTime{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.OpenHour() != input.OpenHour() || output.CloseHour() != input.CloseHour() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
