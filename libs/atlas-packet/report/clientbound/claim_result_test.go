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
