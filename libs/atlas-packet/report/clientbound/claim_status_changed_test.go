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
