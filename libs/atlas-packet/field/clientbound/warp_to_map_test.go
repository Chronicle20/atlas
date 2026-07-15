package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/sirupsen/logrus"
)

// TestWarpToMapByteOutputV79 pins the gms_v79 SET_FIELD warp (bCharacterData=0)
// clientbound wire. IDA: CStage::OnSetField @0x6f07d9 (GMS_v79_1_DEVM.exe),
// else-branch (bCharacterData==0) —
//
//	Decode4(channelId)          @0x6f080c → channel id.
//	Decode1(sNotifierMessage)   @0x6f082b → notifier byte.
//	Decode1(bCharacterData=0)   @0x6f0838 → flag (warp path).
//	Decode2(nNotifierCheck)     @0x6f084f → notifier count (0).
//	Decode4(dwPosMap)           @0x6f0997 → target map id (NO revive byte before it).
//	Decode1(nPortal)            @0x6f09b5 → portal id.
//	Decode2(nHP)                @0x6f09c6 → hp (2 bytes; GMS<95).
//	Decode1(m_bChaseEnable)     @0x6f09de → chase flag (false here).
//	DecodeBuffer(8)             @0x6f0a76 → 8-byte timestamp.
//
// Unlike v83 (which reads a revive Decode1 between nNotifierCheck and mapId),
// v79 has NO revive byte — the codec gates it to >=83. Envelope = 24 bytes.
func TestWarpToMapByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := WarpToMap{channelId: 1, mapId: 100000000, portalId: 0, hp: 500, timestamp: 116444736000000000}
	expected := []byte{
		0x01, 0x00, 0x00, 0x00, // channelId=1 @0x6f080c
		0x00,       // sNotifierMessage @0x6f082b
		0x00,       // bCharacterData=0 @0x6f0838
		0x00, 0x00, // nNotifierCheck=0 @0x6f084f
		0x00, 0xE1, 0xF5, 0x05, // mapId=100000000 @0x6f0997 (no revive before it)
		0x00,       // portalId=0 @0x6f09b5
		0xF4, 0x01, // hp=500 (Decode2) @0x6f09c6
		0x00,                                           // chase=false @0x6f09de
		0x00, 0x80, 0x3E, 0xD5, 0xDE, 0xB1, 0x9D, 0x01, // timestamp int64-LE @0x6f0a76
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 warp_to_map golden mismatch: got %v want %v", actual, expected)
	}
}

// TestWarpToMapByteOutputV72 pins the gms_v72 SET_FIELD warp (bCharacterData=0)
// clientbound wire. IDA: CStage::OnSetField @0x6c0c9b (GMS_v72.1_U_DEVM.exe),
// else-branch (bCharacterData==0) —
//
//	Decode4(channelId)          @0x6c0cce → channel id.
//	Decode1(sNotifierMessage)   @0x6c0ced → notifier byte.
//	Decode1(bCharacterData=0)   @0x6c0cfa → flag (warp path).
//	Decode2(nNotifierCheck)     @0x6c0d11 → notifier count (0).
//	Decode4(dwPosMap)           @0x6c0e59 → target map id (NO revive byte before it).
//	Decode1(nPortal)            @0x6c0e77 → portal id.
//	Decode2(nHP)                @0x6c0e88 → hp (2 bytes; GMS<95).
//	Decode1(m_bChaseEnable)     @0x6c0ea0 → chase flag (false here).
//	DecodeBuffer(8)             @0x6c0f38 → 8-byte timestamp.
//
// Like v79 (and unlike v83, which reads a revive Decode1 between nNotifierCheck
// and mapId), v72 has NO revive byte — the codec gates it to GMS>=83. Every field
// cites a decompile line (this is NOT an opaque family; the warp else-branch reads
// scalar fields, not the CharacterData blob). Envelope = 24 bytes.
func TestWarpToMapByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := WarpToMap{channelId: 1, mapId: 100000000, portalId: 0, hp: 500, timestamp: 116444736000000000}
	expected := []byte{
		0x01, 0x00, 0x00, 0x00, // channelId=1 @0x6c0cce
		0x00,       // sNotifierMessage @0x6c0ced
		0x00,       // bCharacterData=0 @0x6c0cfa
		0x00, 0x00, // nNotifierCheck=0 @0x6c0d11
		0x00, 0xE1, 0xF5, 0x05, // mapId=100000000 @0x6c0e59 (no revive before it)
		0x00,       // portalId=0 @0x6c0e77
		0xF4, 0x01, // hp=500 (Decode2) @0x6c0e88
		0x00,                                           // chase=false @0x6c0ea0
		0x00, 0x80, 0x3E, 0xD5, 0xDE, 0xB1, 0x9D, 0x01, // timestamp int64-LE @0x6c0f38
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 warp_to_map golden mismatch: got %v want %v", actual, expected)
	}
}

// TestWarpToMapByteOutputV61 pins the gms_v61 SET_FIELD warp else-branch (op 0x5C
// = 92). IDA: CStage::OnSetField @0x659fd3 (GMS_v61.1_U_DEVM.exe) bCharacterData=0
// branch reads Decode4(mapId) + Decode1(portalId) + Decode2(hp) + Decode1(chase);
// no revive byte (gated GMS>=83, v61<83), hp is Decode2 (v61<95). The trailing
// DecodeBuffer(8) timestamp closes both branches. Envelope = 24 bytes, byte-
// identical to v72. This clientbound fixture completes the CStage::OnSetField
// worst-of family so the SET_FIELD v61 cell can promote.
// packet-audit:verify packet=field/clientbound/FieldWarpToMap version=gms_v61 ida=0x659fd3
func TestWarpToMapByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := WarpToMap{channelId: 1, mapId: 100000000, portalId: 0, hp: 500, timestamp: 116444736000000000}
	expected := []byte{
		0x01, 0x00, 0x00, 0x00, // channelId=1
		0x00,       // sNotifierMessage
		0x00,       // bCharacterData=0
		0x00, 0x00, // nNotifierCheck=0
		0x00, 0xE1, 0xF5, 0x05, // mapId=100000000 (no revive before it)
		0x00,       // portalId=0
		0xF4, 0x01, // hp=500 (Decode2)
		0x00,                                           // chase=false
		0x00, 0x80, 0x3E, 0xD5, 0xDE, 0xB1, 0x9D, 0x01, // timestamp int64-LE
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v61 warp_to_map golden mismatch: got %v want %v", actual, expected)
	}
}

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
// packet-audit:verify packet=field/clientbound/FieldWarpToMap version=gms_v79 ida=0x6f07d9
// packet-audit:verify packet=field/clientbound/FieldWarpToMap version=jms_v185 ida=0x7eea69
// packet-audit:verify packet=field/clientbound/FieldWarpToMap version=gms_v83 ida=0x776020
// packet-audit:verify packet=field/clientbound/FieldWarpToMap version=gms_v87 ida=0x7c429c
// packet-audit:verify packet=field/clientbound/FieldWarpToMap version=gms_v95 ida=0x71a0a0
// packet-audit:verify packet=field/clientbound/FieldWarpToMap version=gms_v84 ida=0x798987
// packet-audit:verify packet=field/clientbound/FieldWarpToMap version=gms_v72 ida=0x6c0c9b
func TestWarpToMapWireLength(t *testing.T) {
	cases := map[string]int{
		// DecodeOpt is gated >83 (present v87+); oldDriverID is gated GMS>=95; hp is 4 bytes only for GMS>=95, else 2 (incl. JMS).
		"GMS v28":  21, // channelId(4)+sNotifier(1)+bCharData(1)+mapId(4)+portal(1)+hp(2)+timestamp(8); no DecodeOpt/nNotifierCheck/revive/chase (gated >28)
		"GMS v83":  25, // v28 + nNotifierCheck(2)+revive(1)+chase(1); no DecodeOpt (gated >=87), hp 2
		"GMS v84":  25, // == v83: DecodeOpt is v87+, not v84 (off-by-one fix, delta §3.1.6)
		"GMS v86":  25, // == v83: still pre-v87, no DecodeOpt
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

// TestWarpToPositionWireShape pins the chase position-warp: on GMS (chase
// branch gated >28) NewWarpToPosition writes chase=true followed by Decode4 x /
// Decode4 y, i.e. 8 bytes more than the chase=false envelope, and round-trips
// the coordinates. This is the SET_FIELD mechanism Mystic Door uses to land the
// user on the linked door (v83 CStage::OnSetField @0x776020).
func TestWarpToPositionWireShape(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	// chase=false baseline for GMS v83 is 25 bytes (see TestWarpToMapWireLength).
	pos := NewWarpToPosition(1, 100000000, 500, 1234, -567)
	b := pos.Encode(logrus.New(), ctx)(nil)
	if len(b) != 33 {
		t.Fatalf("position-warp length: got %d, want 33 (v83 25 + chase x/y 8)", len(b))
	}

	out := WarpToMap{}
	pt.RoundTrip(t, ctx, pos.Encode, out.Decode, nil)
	if !out.Chase() {
		t.Fatal("chase flag not round-tripped")
	}
	if out.ChaseX() != 1234 || out.ChaseY() != -567 {
		t.Fatalf("chase position: got (%d,%d), want (1234,-567)", out.ChaseX(), out.ChaseY())
	}
	if out.PortalId() != chasePortalId {
		t.Fatalf("position-warp portalId: got %d, want %d (chasePortalId)", out.PortalId(), chasePortalId)
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
