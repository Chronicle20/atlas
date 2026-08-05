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
