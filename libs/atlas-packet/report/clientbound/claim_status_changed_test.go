package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestClaimSvrStatusChangedGolden(t *testing.T) {
	// 1 byte connected flag; nonzero sets m_bClaimSvrConnected.
	input := NewClaimSvrStatusChanged(true)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimSvrStatusChangedByteOutputV72 verifies the wire-exact byte output
// of ClaimSvrStatusChanged for GMS v72.
// IDA evidence (session c8acae95, GMS_v72.1_U_DEVM.exe.i64):
//
//	CWvsContext::OnClaimSvrStatusChanged@0x91fc79, resolved via the opcode
//	dispatch table CWvsContext::OnPacket case 44 @0x902736 (registry
//	gms_v72.yaml op CLAIM_STATUS_CHANGED, opcode 44/0x2C). CInPacket::Decode1(a2)
//	@0x91fc89 reads a single byte, compared != 0 to produce a bool, stored at
//	this+11892 (m_bClaimSvrConnected). No further reads. 1 byte total.
//
// packet-audit:verify packet=report/clientbound/ClaimSvrStatusChanged version=gms_v72 ida=0x91fc79
func TestClaimSvrStatusChangedByteOutputV72(t *testing.T) {
	v := pt.Variants[9] // GMS v72
	if v.Name != "GMS v72" {
		t.Fatalf("pt.Variants[9] = %q, want %q (index drifted)", v.Name, "GMS v72")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimSvrStatusChanged(true)
	expected := []byte{0x01} // Decode1 connected @0x91fc89
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimSvrStatusChangedByteOutputV79 verifies the wire-exact byte output
// of ClaimSvrStatusChanged for GMS v79.
// IDA evidence (session 1438cecd, GMS_v79_1_DEVM.exe.i64):
//
//	CWvsContext::OnClaimSvrStatusChanged@0x971bc4, resolved via the opcode
//	dispatch table CWvsContext::OnPacket case 44 @0x95396e (registry
//	gms_v79.yaml op CLAIM_STATUS_CHANGED, opcode 44/0x2C). CInPacket::Decode1(a2)
//	@0x971bd4 reads a single byte, compared != 0 to produce a bool, stored at
//	this+12120 (m_bClaimSvrConnected). No further reads. 1 byte total.
//
// packet-audit:verify packet=report/clientbound/ClaimSvrStatusChanged version=gms_v79 ida=0x971bc4
func TestClaimSvrStatusChangedByteOutputV79(t *testing.T) {
	v := pt.Variants[10] // GMS v79
	if v.Name != "GMS v79" {
		t.Fatalf("pt.Variants[10] = %q, want %q (index drifted)", v.Name, "GMS v79")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimSvrStatusChanged(true)
	expected := []byte{0x01} // Decode1 connected @0x971bd4
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimSvrStatusChangedByteOutputV83 verifies the wire-exact byte output
// of ClaimSvrStatusChanged for GMS v83.
// IDA evidence (session 41f13e0d, v83_Me MapleStory_dump.exe.i64):
//
//	CWvsContext::OnClaimSvrStatusChanged@0xa27b61, resolved via the opcode
//	dispatch table CWvsContext::OnPacket case 0x2F @0xa07b76 (registry
//	gms_v83.yaml op CLAIM_STATUS_CHANGED, opcode 47/0x2F). CInPacket::Decode1(a2)
//	@0xa27b73 reads a single byte, compared != 0 to produce a bool, stored at
//	this[3120] (m_bClaimSvrConnected). No further reads. 1 byte total.
//	Byte-identical to the v72/v79 twins.
//
// packet-audit:verify packet=report/clientbound/ClaimSvrStatusChanged version=gms_v83 ida=0xa27b61
func TestClaimSvrStatusChangedByteOutputV83(t *testing.T) {
	v := pt.Variants[1] // GMS v83
	if v.Name != "GMS v83" {
		t.Fatalf("pt.Variants[1] = %q, want %q (index drifted)", v.Name, "GMS v83")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimSvrStatusChanged(true)
	expected := []byte{0x01} // Decode1 connected @0xa27b73
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimSvrStatusChangedByteOutputV84 verifies the wire-exact byte output
// of ClaimSvrStatusChanged for GMS v84.
// IDA evidence (session 5881cf84, GMS_v84.1_U_DEVM.i64):
//
//	CWvsContext::OnClaimSvrStatusChanged@0xa7331c (named this pass; was
//	sub_A7331C), resolved via the opcode dispatch table
//	CWvsContext::OnPacket@0xa51cd0 case 0x2F, call-site @0xa51e4b (registry
//	gms_v84.yaml op CLAIM_STATUS_CHANGED, opcode 47/0x2F -- confirmed
//	identical to v83, not shifted). CInPacket::Decode1(a2) @0xa7332c reads a
//	single byte, compared != 0 to produce a bool, stored this[3140]
//	(m_bClaimSvrConnected). No further reads. 1 byte total. Byte-identical
//	to the v72/v79/v83 twins.
//
// packet-audit:verify packet=report/clientbound/ClaimSvrStatusChanged version=gms_v84 ida=0xa7331c
func TestClaimSvrStatusChangedByteOutputV84(t *testing.T) {
	v := pt.Variants[5] // GMS v84
	if v.Name != "GMS v84" {
		t.Fatalf("pt.Variants[5] = %q, want %q (index drifted)", v.Name, "GMS v84")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimSvrStatusChanged(true)
	expected := []byte{0x01} // Decode1 connected @0xa7332c
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimSvrStatusChangedByteOutputV87 verifies the wire-exact byte output
// of ClaimSvrStatusChanged for GMS v87.
// IDA evidence (session d51ecbd3, GMSv87_4GB.exe.i64):
//
//	CWvsContext::OnClaimSvrStatusChanged@0xabf7ce, resolved via the opcode
//	dispatch table CWvsContext::OnPacket@0xa9d011 case 0x2F @0xa9d18c
//	(registry gms_v87.yaml op CLAIM_STATUS_CHANGED, opcode 47/0x2F --
//	STATUS.md's pre-filled v87 column value of 0x030 is stale/wrong;
//	independently re-derived here from the live dispatch switch, which
//	agrees with the registry). CInPacket::Decode1(a2) @0xabf7e0 reads a
//	single byte, compared != 0 to produce a bool, stored this[3163]
//	(m_bClaimSvrConnected). No further reads. 1 byte total. Byte-identical
//	to the v72/v79/v83/v84 twins.
//
// packet-audit:verify packet=report/clientbound/ClaimSvrStatusChanged version=gms_v87 ida=0xabf7ce
func TestClaimSvrStatusChangedByteOutputV87(t *testing.T) {
	v := pt.Variants[2] // GMS v87
	if v.Name != "GMS v87" {
		t.Fatalf("pt.Variants[2] = %q, want %q (index drifted)", v.Name, "GMS v87")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimSvrStatusChanged(true)
	expected := []byte{0x01} // Decode1 connected @0xabf7e0
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimSvrStatusChangedByteOutputV92 verifies the wire-exact byte output
// of ClaimSvrStatusChanged for GMS v92.
// IDA evidence (session acdfccff, GMS_v92_1_DEVM.exe.i64):
//
//	CWvsContext::OnClaimSvrStatusChanged@0x9c5d60, resolved via the opcode
//	dispatch table CWvsContext::OnPacket@0x9ba740 case 48 @0x9ba8c8
//	(registry gms_v92.yaml op CLAIM_STATUS_CHANGED, opcode 48/0x30 --
//	matches STATUS.md's pre-filled v92 column value of 0x030, and matches
//	the live dispatch switch, independently re-derived here).
//	CInPacket::Decode1(a2) @0x9c5d67 reads a single byte, compared != 0 to
//	produce a bool, stored this+13896 (m_bClaimSvrConnected). No further
//	reads. 1 byte total. Byte-identical to the v72/v79/v83/v84/v87 twins.
//
// packet-audit:verify packet=report/clientbound/ClaimSvrStatusChanged version=gms_v92 ida=0x9c5d60
func TestClaimSvrStatusChangedByteOutputV92(t *testing.T) {
	v := pt.Variants[11] // GMS v92
	if v.Name != "GMS v92" {
		t.Fatalf("pt.Variants[11] = %q, want %q (index drifted)", v.Name, "GMS v92")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimSvrStatusChanged(true)
	expected := []byte{0x01} // Decode1 connected @0x9c5d67
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimSvrStatusChangedByteOutputV95 verifies the wire-exact byte output
// of ClaimSvrStatusChanged for GMS v95.
// IDA evidence (session 79906a1e, GMS_v95.0_U_DEVM.exe.i64):
//
//	CWvsContext::OnClaimSvrStatusChanged@0x9f1650, resolved via the opcode
//	dispatch table CWvsContext::OnPacket@0x9e5830 case 46, call-site
//	@0x9e59b8 (registry gms_v95.yaml op CLAIM_STATUS_CHANGED, opcode
//	46/0x2E -- matches STATUS.md's pre-filled v95 column value of 0x02E).
//	Single CInPacket::Decode1(iPacket) != 0 @0x9f1663, stored directly to
//	this->m_bClaimSvrConnected -- 1 byte total, no branches. Byte-identical
//	to the v72/v79/v83/v84/v87/v92 twins.
//
// packet-audit:verify packet=report/clientbound/ClaimSvrStatusChanged version=gms_v95 ida=0x9f1650
func TestClaimSvrStatusChangedByteOutputV95(t *testing.T) {
	v := pt.Variants[3] // GMS v95
	if v.Name != "GMS v95" {
		t.Fatalf("pt.Variants[3] = %q, want %q (index drifted)", v.Name, "GMS v95")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimSvrStatusChanged(true)
	expected := []byte{0x01} // Decode1 connected @0x9f1663
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimSvrStatusChangedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClaimSvrStatusChanged(true)
			output := ClaimSvrStatusChanged{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Connected() != input.Connected() {
				t.Errorf("round-trip mismatch: got %v want %v", output.Connected(), input.Connected())
			}
		})
	}
}
