package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestAllCharacterListPongRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AllCharacterListPong{render: true}
			output := AllCharacterListPong{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Render() != input.Render() {
				t.Errorf("render: got %v, want %v", output.Render(), input.Render())
			}
		})
	}
}
