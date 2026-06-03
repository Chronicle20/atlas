package serverbound

import (
	"testing"

	channel2 "github.com/Chronicle20/atlas/libs/atlas-constants/channel"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
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

// TestChannelChangeRequestWireShape proves the exact wire layout matches what
// CField::SendTransferChannelRequest (GMS v95 @ 0x52efa0) encodes:
//
//	Encode1 (nTargetChannel) + Encode4 (get_update_time()) = 5 bytes.
//
// All versions share the same layout — no version gate needed.
func TestChannelChangeRequestWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := ChannelChangeRequest{channelId: channel2.Id(3), updateTime: 99999}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// 1 (channelId byte) + 4 (updateTime uint32 LE) = 5 bytes
			if len(b) != 5 {
				t.Errorf("wire size = %d bytes, want 5: % x", len(b), b)
			}
			// Byte 0 must be the channel ID
			if b[0] != byte(in.channelId) {
				t.Errorf("byte[0] = 0x%02x, want 0x%02x (channelId)", b[0], byte(in.channelId))
			}
			// Bytes 1-4: updateTime = 99999 = 0x0001869F, LE → 9F 86 01 00
			wantLE := [4]byte{0x9F, 0x86, 0x01, 0x00}
			if b[1] != wantLE[0] || b[2] != wantLE[1] || b[3] != wantLE[2] || b[4] != wantLE[3] {
				t.Errorf("updateTime bytes = % x, want % x (99999 LE)", b[1:5], wantLE)
			}
		})
	}
}
