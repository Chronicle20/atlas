package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestMultiChatRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := MultiChat{mode: 1, from: "PlayerOne", message: "party chat message"}
			output := MultiChat{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.From() != input.From() {
				t.Errorf("from: got %v, want %v", output.From(), input.From())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}
