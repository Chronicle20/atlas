package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestServerListEntryRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerListEntry{
				worldId:      3,
				worldName:    "Scania",
				state:        1,
				eventMessage: "Welcome",
				channelLoads: []model.ChannelLoad{
					model.NewChannelLoad(1, 100),
					model.NewChannelLoad(2, 200),
				},
			}
			output := ServerListEntry{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.WorldId() != input.WorldId() {
				t.Errorf("worldId: got %v, want %v", output.WorldId(), input.WorldId())
			}
			if output.WorldName() != input.WorldName() {
				t.Errorf("worldName: got %v, want %v", output.WorldName(), input.WorldName())
			}
			if len(output.ChannelLoads()) != len(input.ChannelLoads()) {
				t.Errorf("channelLoads length: got %v, want %v", len(output.ChannelLoads()), len(input.ChannelLoads()))
			}
		})
	}
}
