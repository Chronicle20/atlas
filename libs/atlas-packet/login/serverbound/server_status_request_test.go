package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/serverbound/ServerStatusRequest version=gms_v83 ida=0x5f8078
// packet-audit:verify packet=login/serverbound/ServerStatusRequest version=gms_v87 ida=0x62f80a
// packet-audit:verify packet=login/serverbound/ServerStatusRequest version=gms_v95 ida=0x5d43d0
// packet-audit:verify packet=login/serverbound/ServerStatusRequest version=gms_v84 ida=0x60cfee
func TestServerStatusRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerStatusRequest{worldId: world.Id(3)}
			output := ServerStatusRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.WorldId() != input.WorldId() {
				t.Errorf("worldId: got %v, want %v", output.WorldId(), input.WorldId())
			}
		})
	}
}
