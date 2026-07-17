package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestTvSendMessageResultSuccessRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewTvSendMessageResultSuccess()
			output := TvSendMessageResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.HasError() {
				t.Errorf("hasError: got true, want false")
			}
		})
	}
}

func TestTvSendMessageResultSuccessByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewTvSendMessageResultSuccess()
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 1 {
		t.Fatalf("payload length: got %d, want 1", len(actual))
	}
	if actual[0] != 0 {
		t.Errorf("payload byte: got %d, want 0", actual[0])
	}
}

func TestTvSendMessageResultErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewTvSendMessageResultError(2)
			output := TvSendMessageResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if !output.HasError() {
				t.Errorf("hasError: got false, want true")
			}
			if output.Code() != 2 {
				t.Errorf("code: got %v, want 2", output.Code())
			}
		})
	}
}
