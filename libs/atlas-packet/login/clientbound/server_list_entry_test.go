package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// gms_v61: WORLD_INFORMATION world-record decoder sub_56663F @0x56663f
// (GMS_v61.1_U_DEVM.exe, port 13338): Decode1(worldId)@0x566660, DecodeStr(name)
// @0x5666c4, Decode1(state)@0x5666f8, DecodeStr(eventDesc)@0x566701, Decode2(exp)
// @0x566737, Decode2(drop)@0x566744, Decode1(blockCharCreation)@0x566751,
// Decode1(channelCount)@0x566754, per-channel loop, Decode2(balloonCount)
// @0x5667ea, per-balloon loop — structure identical to the verified v79 cell,
// matching atlas ServerListEntry.Encode (GMS>12 path).
//
// packet-audit:verify packet=login/clientbound/ServerListEntry version=gms_v61 ida=0x56663f
func TestServerListEntryV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	// Minimal entry (no channels, no balloons) → deterministic wire:
	//   worldId(1) + name "W"(2+1) + state(1) + eventMsg ""(2) + exp 100(2) +
	//   drop 100(2) + blockCreation(1) + channelCount 0(1) + balloonCount 0(2).
	input := ServerListEntry{worldId: world.Id(1), worldName: "W"}
	want := []byte{
		0x01, 0x01, 0x00, 'W', // worldId, name len+bytes
		0x00, 0x00, 0x00, // state, eventMessage (len 0)
		0x64, 0x00, 0x64, 0x00, // exp 100, drop 100
		0x00,       // block char creation
		0x00,       // channel count
		0x00, 0x00, // balloon count
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 ServerListEntry body: got % x, want % x", got, want)
	}
}

// gms_v72: WORLD_INFORMATION (op 10) world-list entries are decoded by
// CLogin::OnWorldInformation = sub_5B33F8 @0x5b33f8 (GMS_v72.1_U_DEVM.exe, port
// 13339); the per-world entry fields (worldId, name, channel loop, balloons) are
// version-stable and round-trip below. Marker-only (tier-0).
// packet-audit:verify packet=login/clientbound/ServerListEntry version=gms_v72 ida=0x5b33f8
// packet-audit:verify packet=login/clientbound/ServerListEntry version=gms_v83 ida=0x5f95b7
// packet-audit:verify packet=login/clientbound/ServerListEntry version=gms_v87 ida=0x630e7c
// packet-audit:verify packet=login/clientbound/ServerListEntry version=gms_v95 ida=0x5da7f0
// packet-audit:verify packet=login/clientbound/ServerListEntry version=gms_v84 ida=0x60e5b3
// packet-audit:verify packet=login/clientbound/ServerListEntry version=jms_v185 ida=0x66f107
// packet-audit:verify packet=login/clientbound/ServerListEntry version=gms_v79 ida=0x5ce248
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
				nil,
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

func TestServerListEntryRoundTripWithBalloons(t *testing.T) {
	for _, v := range pt.Variants {
		// Balloon block is only emitted for (GMS && >12) || JMS.
		if !((v.Region == "GMS" && v.MajorVersion > 12) || v.Region == "JMS") {
			continue
		}
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			balloons := []model.WorldBalloon{
				model.NewWorldBalloon(10, 20, "Event running!"),
				model.NewWorldBalloon(300, 400, "Maintenance Friday"),
			}
			input := NewServerListEntry(
				3, "Scania", 1, "Welcome",
				[]model.ChannelLoad{model.NewChannelLoad(1, 100)},
				balloons,
			)
			output := ServerListEntry{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Balloons()) != 2 {
				t.Fatalf("balloons length: got %d, want 2", len(output.Balloons()))
			}
			if output.Balloons()[0].Message() != "Event running!" {
				t.Errorf("balloon[0] message: got %q", output.Balloons()[0].Message())
			}
			if output.Balloons()[1].X() != 300 || output.Balloons()[1].Y() != 400 {
				t.Errorf("balloon[1] pos: got (%d,%d)", output.Balloons()[1].X(), output.Balloons()[1].Y())
			}
		})
	}
}
