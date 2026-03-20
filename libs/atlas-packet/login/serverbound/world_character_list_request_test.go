package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestWorldCharacterListRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WorldCharacterListRequest{
				gameStartMode: 1,
				worldId:       world.Id(2),
				channelId:     channel.Id(3),
				socketAddr:    12345,
			}
			output := WorldCharacterListRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if v.Region == "GMS" && v.MajorVersion > 28 {
				if output.GameStartMode() != input.GameStartMode() {
					t.Errorf("gameStartMode: got %v, want %v", output.GameStartMode(), input.GameStartMode())
				}
			}
			if output.WorldId() != input.WorldId() {
				t.Errorf("worldId: got %v, want %v", output.WorldId(), input.WorldId())
			}
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
			if (v.Region == "GMS" && v.MajorVersion > 12) || v.Region == "JMS" {
				if output.SocketAddr() != input.SocketAddr() {
					t.Errorf("socketAddr: got %v, want %v", output.SocketAddr(), input.SocketAddr())
				}
			}
		})
	}
}
