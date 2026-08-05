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
