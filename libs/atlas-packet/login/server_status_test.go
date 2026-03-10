package login

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestServerStatusRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerStatus{status: 1}
			output := ServerStatus{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Status() != input.Status() {
				t.Errorf("status: got %v, want %v", output.Status(), input.Status())
			}
		})
	}
}
