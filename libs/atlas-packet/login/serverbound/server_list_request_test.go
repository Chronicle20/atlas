package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestServerListRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerListRequest{}
			output := ServerListRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
