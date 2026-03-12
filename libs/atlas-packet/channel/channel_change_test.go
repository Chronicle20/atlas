package channel

import (
	"testing"

	channel2 "github.com/Chronicle20/atlas-constants/channel"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestChannelChangeRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ChannelChangeRequest{channelId: channel2.Id(5), updateTime: 12345}
			output := ChannelChangeRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
