package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestServerListEntryWorldIdInChannels(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			const wantWorldId byte = 3
			input := NewServerListEntry(
				world.Id(wantWorldId), "TestWorld", 0, "",
				[]model.ChannelLoad{
					model.NewChannelLoad(channel.Id(1), 100),
					model.NewChannelLoad(channel.Id(2), 100),
				},
			)
			l, _ := testlog.NewNullLogger()
			bytes := input.Encode(l, ctx)(nil)

			req := request.Request(bytes)
			r := request.NewRequestReader(&req, 0)

			// Skip top-level header — match the encode order
			_ = r.ReadByte()        // worldId
			_ = r.ReadAsciiString() // worldName
			if v.Region == "GMS" {
				if v.MajorVersion > 12 {
					_ = r.ReadByte()        // state
					_ = r.ReadAsciiString() // eventMessage
					_ = r.ReadUint16()      // expRate
					_ = r.ReadUint16()      // dropRate
					_ = r.ReadByte()        // block char creation
				}
			} else if v.Region == "JMS" {
				_ = r.ReadByte()
				_ = r.ReadAsciiString()
				_ = r.ReadUint16()
				_ = r.ReadUint16()
			}
			channelCount := r.ReadByte()
			for i := byte(0); i < channelCount; i++ {
				_ = r.ReadAsciiString() // channel name
				_ = r.ReadUint32()      // capacity
				gotWorldId := r.ReadByte()
				if gotWorldId != wantWorldId {
					t.Errorf("channel %d worldId byte: got %d, want %d", i, gotWorldId, wantWorldId)
				}
				_ = r.ReadByte() // channelId
				_ = r.ReadBool() // adult channel
			}
		})
	}
}

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
