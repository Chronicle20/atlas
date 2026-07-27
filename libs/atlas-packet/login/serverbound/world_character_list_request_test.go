package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/serverbound/WorldCharacterListRequest version=gms_v83 ida=0x5f6d6a
// packet-audit:verify packet=login/serverbound/WorldCharacterListRequest version=gms_v87 ida=0x62e463
// packet-audit:verify packet=login/serverbound/WorldCharacterListRequest version=gms_v95 ida=0x5dbef0
// packet-audit:verify packet=login/serverbound/WorldCharacterListRequest version=gms_v84 ida=0x60bca3
// packet-audit:verify packet=login/serverbound/WorldCharacterListRequest version=jms_v185 ida=0x66db89
// packet-audit:verify packet=login/serverbound/WorldCharacterListRequest version=gms_v79 ida=0x5cc905
//
// gms_v72: CLogin::SendLoginPacket = sub_5B1B25 @0x5b1b25 (GMS_v72.1_U_DEVM.exe,
// port 13339): COutPacket(5) @0x5b1c45; Encode1(worldId) @0x5b1c56; Encode1(
// channelId) @0x5b1c61; Encode4(socketAddr=getsockname) @0x5b1c92. NO
// gameStartMode byte (v72<83) — matches the codec's >=83 gate. socketAddr present
// (>12). Wire = worldId + channelId + socketAddr = 6 bytes.
//
// packet-audit:verify packet=login/serverbound/WorldCharacterListRequest version=gms_v72 ida=0x5b1b25
func TestWorldCharacterListRequestV72Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := WorldCharacterListRequest{gameStartMode: 1, worldId: world.Id(2), channelId: channel.Id(3), socketAddr: 12345}
	want := []byte{0x02, 0x03, 0x39, 0x30, 0x00, 0x00} // worldId, channelId, socketAddr LE (no gameStartMode)
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v72 WorldCharacterListRequest body: got % x, want % x", got, want)
	}
}

// gms_v61: CLogin::SendLoginPacket twin = sub_564DC9 @0x564dc9 (GMS_v61.1_U_DEVM.exe,
// port 13338): COutPacket(5) @0x564eeb; Encode1(worldId=*(BYTE*)v9) @0x564efc;
// Encode1(channelId=a3) @0x564f07; SendPacket @0x564f16. NO gameStartMode byte
// (v61<83) and — unlike v72 — NO getsockname/Encode4(socketAddr): the v72 twin
// sub_5B1B25@0x5b1b25 adds getsockname->Encode4@0x5b1c92, absent here. Wire =
// worldId + channelId = 2 bytes only.
//
// packet-audit:verify packet=login/serverbound/WorldCharacterListRequest version=gms_v61 ida=0x564dc9
func TestWorldCharacterListRequestV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := WorldCharacterListRequest{gameStartMode: 1, worldId: world.Id(2), channelId: channel.Id(3), socketAddr: 12345}
	want := []byte{0x02, 0x03} // worldId, channelId (no gameStartMode, no socketAddr)
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 WorldCharacterListRequest body: got % x, want % x", got, want)
	}
}

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
			// gameStartMode exists only at GMS v83+ (IDA v79 SendLoginPacket@0x5cc905 omits it).
			if v.Region == "GMS" && v.MajorVersion >= 83 {
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
			// socketAddr int is a v72+ addition (IDA v61 sub_564DC9@0x564dc9 omits it).
			if (v.Region == "GMS" && v.MajorVersion >= 72) || v.Region == "JMS" {
				if output.SocketAddr() != input.SocketAddr() {
					t.Errorf("socketAddr: got %v, want %v", output.SocketAddr(), input.SocketAddr())
				}
			}
		})
	}
}
