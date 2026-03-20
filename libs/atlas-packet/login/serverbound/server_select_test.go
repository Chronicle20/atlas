package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas-constants/world"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestServerSelectRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerSelect{worldId: world.Id(3)}
			output := ServerSelect{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.WorldId() != input.WorldId() {
				t.Errorf("worldId: got %v, want %v", output.WorldId(), input.WorldId())
			}
		})
	}
}
