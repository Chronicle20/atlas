package socket

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestPongRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Pong{}
			output := Pong{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
