package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestEnterDoorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			in := Enter{ownerId: 4242, direction: 1}
			out := Enter{}
			pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
			if out.OwnerId() != in.OwnerId() || out.Direction() != in.Direction() {
				t.Fatalf("roundtrip mismatch: %+v vs %+v", out, in)
			}
		})
	}
}
