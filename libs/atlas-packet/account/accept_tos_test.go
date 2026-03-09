package account

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestAcceptTosRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AcceptTos{accepted: true}
			output := AcceptTos{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Accepted() != input.Accepted() {
				t.Errorf("accepted: got %v, want %v", output.Accepted(), input.Accepted())
			}
		})
	}
}
