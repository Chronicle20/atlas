package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestLoginAuthRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := LoginAuth{screen: "MapLogin"}
			output := LoginAuth{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Screen() != input.Screen() {
				t.Errorf("screen: got %v, want %v", output.Screen(), input.Screen())
			}
		})
	}
}
