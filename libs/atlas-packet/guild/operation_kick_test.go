package guild

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestKickRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Kick{cid: 67890, name: "BadPlayer"}
			output := Kick{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Cid() != input.Cid() {
				t.Errorf("cid: got %v, want %v", output.Cid(), input.Cid())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}
