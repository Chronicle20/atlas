package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestServerListEntryBytesV48 pins the gms_v48 world-info wire.
//
// IDA (GMS_v48_1_DEVM.exe): CLogin::OnWorldInformation @0x50120a reads
// Decode1(worldId) @0x501225. On the >= 0 arm — the entry form — it then reads:
//
//	DecodeStr           → worldName
//	Decode1  @0x501306  → state
//	DecodeStr           → eventMessage
//	Decode2  @0x50133b  → eventExpRate
//	Decode2  @0x501348  → eventDropRate
//	Decode1  @0x501355  → blockCharCreation
//	Decode1  @0x50135d  → channelCount
//	per channel: DecodeStr, Decode4 @0x5013af, Decode1 @0x5013bc,
//	             Decode1 @0x5013c9, Decode1
//
// and then returns at 0x5013dc — no balloon block. v61 @0x56663f reads the same
// prefix plus Decode2 @0x5667ea (balloon count) and a {Decode2, Decode2,
// DecodeStr} loop, so the block arrived between v48 and v61 and was previously
// gated on MajorVersion() > 12.
//
// packet-audit:verify packet=login/clientbound/ServerListEntry version=gms_v48 ida=0x50120a
func TestServerListEntryBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)

	in := NewServerListEntry(0, "Scania", 0, "", []model.ChannelLoad{
		model.NewChannelLoad(channel.Id(1), 100),
	}, nil)
	got := in.Encode(l, ctx)(nil)

	want := []byte{
		0x00,                                     // worldId 0            — Decode1 @0x501225
		0x06, 0x00, 'S', 'c', 'a', 'n', 'i', 'a', // worldName "Scania" — DecodeStr
		0x00,       // state 0              — Decode1 @0x501306
		0x00, 0x00, // eventMessage ""      — DecodeStr
		0x64, 0x00, // eventExpRate 100     — Decode2 @0x50133b
		0x64, 0x00, // eventDropRate 100    — Decode2 @0x501348
		0x00,                                                         // blockCharCreation 0  — Decode1 @0x501355
		0x01,                                                         // channelCount 1       — Decode1 @0x50135d
		0x0A, 0x00, 'S', 'c', 'a', 'n', 'i', 'a', ' ', '-', ' ', '2', // channel name (retail 1-based label; channel.Id(1) -> "2")
		0x64, 0x00, 0x00, 0x00, // capacity 100         — Decode4 @0x5013af
		0x00, // per-channel worldId  — Decode1 @0x5013bc
		0x00, // channelId - 1        — Decode1 @0x5013c9
		0x00, // adult channel        — Decode1
		// no balloon short: absent until v61.
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 ServerListEntry wire:\n got % x\nwant % x", got, want)
	}
}

// TestServerListEntryBalloonBoundary guards the v48/v61 boundary: v48 stops after
// the channel loop, v61 onward appends the balloon count short.
func TestServerListEntryBalloonBoundary(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewServerListEntry(0, "Scania", 0, "", []model.ChannelLoad{
		model.NewChannelLoad(channel.Id(1), 100),
	}, nil)

	v48 := in.Encode(l, pt.CreateContext("GMS", 48, 1))(nil)
	for _, v := range []uint16{61, 72, 79, 83, 87, 95} {
		got := in.Encode(l, pt.CreateContext("GMS", v, 1))(nil)
		if len(got) != len(v48)+2 {
			t.Errorf("GMS v%d length = %d, want %d (v48 + balloon count)", v, len(got), len(v48)+2)
			continue
		}
		if !bytes.Equal(got[:len(v48)], v48) {
			t.Errorf("GMS v%d diverges from v48 before the balloon block", v)
		}
		if !bytes.Equal(got[len(v48):], []byte{0x00, 0x00}) {
			t.Errorf("GMS v%d balloon count = % x, want 00 00", v, got[len(v48):])
		}
	}
}
