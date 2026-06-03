package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/sirupsen/logrus"
)

// TestWarpToMapWireLength pins the exact encoded envelope length per version,
// proving (a) m_dwOldDriverID (4 bytes) is present only on GMS v95+ and (b) nHP
// is 2 bytes on GMS v83/v87 vs 4 bytes on GMS v95+/JMS.
//
// GMS v83/v87 envelope: channelId(4) + sNotifier(1) + bCharData(1) +
//   nNotifierCheck(2) + revive(1) + mapId(4) + portal(1) + hp(2) + chase(1) +
//   timestamp(8) = 25 bytes.
// GMS v95 adds DecodeOpt(2) + oldDriverID(4) and widens hp 2→4 => 25+2+4+2 = 33.
// JMS adds DecodeOpt(2) + JMS pair(5) but has NO chase byte (gated GMS only) and
// hp stays 2 (JMS185 @0x7eec9d Decode2) => 25 - chase(1) + 2 + 5 = 31.
func TestWarpToMapWireLength(t *testing.T) {
	cases := map[string]int{
		// DecodeOpt is gated >83 (present v87+); oldDriverID is gated GMS>=95; hp is 4 bytes only for GMS>=95, else 2 (incl. JMS).
		"GMS v28":  21, // channelId(4)+sNotifier(1)+bCharData(1)+mapId(4)+portal(1)+hp(2)+timestamp(8); no DecodeOpt/nNotifierCheck/revive/chase (gated >28)
		"GMS v83":  25, // v28 + nNotifierCheck(2)+revive(1)+chase(1); no DecodeOpt (gated >83), hp 2
		"GMS v87":  27, // v83 + DecodeOpt(2); still no oldDriverID (gated >=95), hp 2
		"GMS v95":  33, // v87 + oldDriverID(4); hp widened 2->4
		"JMS v185": 31, // v83(25) - chase(1) + DecodeOpt(2)+JMSpair(5); no oldDriverID (GMS-only); hp stays 2 (JMS185 @0x7eec9d Decode2)
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WarpToMap{channelId: 1, mapId: 100000000, portalId: 0, hp: 500, timestamp: 116444736000000000}
			b := input.Encode(logrus.New(), ctx)(nil)
			want, ok := cases[v.Name]
			if !ok {
				t.Fatalf("no expected length for variant %s", v.Name)
			}
			if len(b) != want {
				t.Errorf("encoded length: got %d, want %d", len(b), want)
			}
		})
	}
}

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
