package field

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestWarpToMapRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WarpToMap{channelId: 1, mapId: 100000000, portalId: 0, hp: 500, timestamp: 116444736000000000}
			output := WarpToMap{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
			if output.MapId() != input.MapId() {
				t.Errorf("mapId: got %v, want %v", output.MapId(), input.MapId())
			}
			if output.PortalId() != input.PortalId() {
				t.Errorf("portalId: got %v, want %v", output.PortalId(), input.PortalId())
			}
			if output.Hp() != input.Hp() {
				t.Errorf("hp: got %v, want %v", output.Hp(), input.Hp())
			}
		})
	}
}
