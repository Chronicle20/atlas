package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestTvClearMessageRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewTvClearMessage()
			output := TvClearMessage{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestTvClearMessageByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewTvClearMessage()
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("payload length: got %d, want 0", len(actual))
	}
}
