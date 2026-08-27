package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestCTSForeignEmptyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterTemporaryStat()
			output := NewCharacterTemporaryStat()
			pt.RoundTrip(t, ctx, input.EncodeForeign, output.DecodeForeign, nil)
			if len(output.stats) != 0 {
				t.Errorf("expected 0 decoded stats, got %d", len(output.stats))
			}
		})
	}
}

func TestCTSForeignSingleStatRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterTemporaryStat()
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeSpeed), 2001002, 20, 10, time.Now().Add(time.Minute))
			output := NewCharacterTemporaryStat()
			pt.RoundTrip(t, ctx, input.EncodeForeign, output.DecodeForeign, nil)
			if len(output.stats) != 1 {
				t.Errorf("expected 1 decoded stat, got %d", len(output.stats))
			}
			if sv, ok := output.stats[character.TemporaryStatTypeSpeed]; ok {
				if sv.Value() != 20 {
					t.Errorf("speed value: got %d, want 20", sv.Value())
				}
			} else {
				t.Error("expected Speed stat to be present")
			}
		})
	}
}

// TestCTSEncodeSlowDiseasePerStatLayout pins the v83 wire bytes for a SLOW
// (mob skill 126 level 2, value 80, duration 15000ms) applied via the self
// Encode path. The v83 client reads the per-stat block as
// (Short value | Short mobSkillId | Short mobSkillLevel | Int duration); the
// older atlas encoder wrote (Short value | Int sourceId | Int duration),
// which sent level=0 in bytes 4-5 and crashed the client's render path on
// MobSkill(126, 0) lookup. This test asserts the corrected per-stat 10 bytes
// match the v83 read order.
func TestCTSEncodeSlowDiseasePerStatLayout(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	input := NewCharacterTemporaryStat()
	// Mob skill 126 (Slow) level 2: amount=80%, duration=15000ms.
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeSlow), 126, 80, 2, time.Now().Add(15*time.Second))

	got := input.Encode(nil, ctx)(nil)

	// Layout: 16 bytes mask + 10 bytes per-stat + trailers.
	if len(got) < 26 {
		t.Fatalf("encoded payload too short: %d bytes", len(got))
	}
	mask, stat := got[:16], got[16:26]

	// Mask: SLOW only. The TwoState base group (EnergyCharge..Undead, v83 shifts
	// 82-88 -> dword[1] 0x01FC0000) is NOT asserted, because this CTS holds none of
	// those stats — claiming RideVehicle here made the client run
	// ShowRideVehicleEffect on a mounted player for an unrelated disease (task-190).
	// Bits map to the wire the same way regardless; this matches the
	// v83 client's flag 1<<(i+82) read from wire bytes 4-7 (IDA SecondaryStat::
	// DecodeForLocal @0x781D0E; UINT128 dword array is big-endian, AND'd in wire order).
	// SLOW (shift 32) lands in dword[2] (wire bytes 8-11) at 0x00000001 -> LE 01 00 00 00.
	wantMask := []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, // no two-state base bits: this CTS holds none
		0x01, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	if !bytes.Equal(mask, wantMask) {
		t.Errorf("mask: got %x want %x", mask, wantMask)
	}

	// Per-stat: Short value=80 (50 00) | Short skill=126 (7E 00) |
	// Short level=2 (02 00) | Int duration ~ 15000 (98 3A 00 00).
	// Duration is computed against time.Now() at encode, so check only the
	// load-bearing first 6 bytes (value | skill | level).
	wantStatHead := []byte{0x50, 0x00, 0x7E, 0x00, 0x02, 0x00}
	if !bytes.Equal(stat[:6], wantStatHead) {
		t.Errorf("per-stat head: got %x want %x (full stat: %x)", stat[:6], wantStatHead, stat)
	}
}

// TestCTSEncodeBuffPerStatLayout pins that non-disease stats (e.g.
// Invincible, a player-cast buff) keep the legacy
// (Short value | Int sourceId | Int duration) per-stat shape. Guards against
// a future change accidentally routing buffs through the disease branch.
func TestCTSEncodeBuffPerStatLayout(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	input := NewCharacterTemporaryStat()
	// Bishop Invincible (skill 2301003), level 20, value 30.
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeInvincible), 2301003, 30, 20, time.Now().Add(5*time.Minute))

	got := input.Encode(nil, ctx)(nil)
	if len(got) < 26 {
		t.Fatalf("encoded payload too short: %d bytes", len(got))
	}
	stat := got[16:26]

	// Per-stat: Short value=30 (1E 00) | Int sourceId=2301003 = 0x231C4B
	// (LE: 4B 1C 23 00) | Int duration (varies). Check first 6 bytes.
	wantStatHead := []byte{0x1E, 0x00, 0x4B, 0x1C, 0x23, 0x00}
	if !bytes.Equal(stat[:6], wantStatHead) {
		t.Errorf("per-stat head: got %x want %x (full stat: %x)", stat[:6], wantStatHead, stat)
	}
}

func TestCTSForeignMultiStatRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterTemporaryStat()
			// Byte writer
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeSpeed), 2001002, 20, 10, time.Now().Add(time.Minute))
			// Int writer
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeStun), 0, 1, 5, time.Now().Add(time.Minute))
			output := NewCharacterTemporaryStat()
			pt.RoundTrip(t, ctx, input.EncodeForeign, output.DecodeForeign, nil)
			if len(output.stats) != 2 {
				t.Errorf("expected 2 decoded stats, got %d", len(output.stats))
			}
		})
	}
}

func TestCTSMonsterRidingBaseStatEncodesVehicleAndSkill(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	input := NewCharacterTemporaryStat()
	// sourceId = skill id (rOption), amount = vehicle/taming-mob item id (nOption).
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeMonsterRiding), 1004, 1902000, 1, time.Now().Add(time.Hour))

	got := input.Encode(nil, ctx)(nil)

	// The Monster Riding base-stat block must contain nOption=1902000 then rOption=1004
	// as consecutive little-endian int32s.
	want := []byte{0xb0, 0x05, 0x1d, 0x00 /* 1902000 */, 0xec, 0x03, 0x00, 0x00 /* 1004 */}
	if !bytes.Contains(got, want) {
		t.Fatalf("Monster Riding base stat missing nOption=1902000,rOption=1004; got % x", got)
	}
}

// TestCTSMonsterRidingV95MaskAndLayout pins the GMS v95 mount GIVE_BUFF layout.
// On v95 the registry enumerates 122 stats before EnergyCharge, so the two-state
// group is bits 122-128 and RideVehicle/MonsterRiding is bit 125 (IDA-verified from
// the v95 client flag initializers; see v95_secondarystat_table.md). Bits 122-125
// live in logical range 96-127 -> wire dword[0] (bytes 0-3): EnergyCharge(122)|
// DashSpeed(123)|DashJump(124)|RideVehicle(125) = 0x3C000000, RideVehicle = 0x20000000.
// The remaining mask dwords are empty, and MonsterRiding is encoded only as a base
// stat (no per-stat block). Total = 16 mask + 2 leading + 4 base blocks (15+15+15+13).
func TestCTSMonsterRidingV95MaskAndLayout(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeMonsterRiding), 1004, 1902000, 1, time.Now().Add(time.Hour))

	got := input.Encode(nil, ctx)(nil)

	// Mask dword[0] (bytes 0-3) = 0x20000000 -> LE 00 00 00 20: RideVehicle@125
	// alone. The other two-state members are absent from this CTS, so their bits
	// (and blocks) are not emitted.
	if !bytes.Equal(got[0:4], []byte{0x00, 0x00, 0x00, 0x20}) {
		t.Fatalf("v95 mask dword[0] should be 0x20000000 (RideVehicle@125 alone); got % x", got[0:4])
	}
	// dwords [1],[2],[3] (bytes 4-15) empty.
	if !bytes.Equal(got[4:16], make([]byte, 12)) {
		t.Fatalf("v95 mask dwords[1..3] should be empty; got % x", got[4:16])
	}
	// No truncated per-stat block: the 2 leading bytes (00 00) follow the mask.
	if got[16] != 0 || got[17] != 0 {
		t.Fatalf("expected 2 leading bytes (00 00) after mask, not a per-stat block; got % x", got[16:20])
	}
	// MonsterRiding base stat carries nOption=1902000, rOption=1004.
	want := []byte{0xb0, 0x05, 0x1d, 0x00, 0xec, 0x03, 0x00, 0x00}
	if !bytes.Contains(got, want) {
		t.Fatalf("v95 RideVehicle base stat (1902000,1004) missing; got % x", got)
	}
	// 16 mask + 2 leading + the ONE base block this CTS holds (MonsterRiding, 13).
	// It used to be 58: EnergyCharge15 + DashSpeed15 + DashJump15 + MonsterRiding13,
	// with the first three emitted as empty placeholders beside mask bits the CTS
	// never held. Blocks are presence-gated now, in lockstep with the mask.
	if len(got) != 16+2+13 {
		t.Fatalf("v95 mount packet length: got %d want %d", len(got), 16+2+13)
	}
}

func TestCTSMonsterRidingForeignEncodesVehicleAndSkill(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeMonsterRiding), 1004, 1902000, 1, time.Now().Add(time.Hour))

	got := input.EncodeForeign(nil, ctx)(nil)

	want := []byte{0xb0, 0x05, 0x1d, 0x00, 0xec, 0x03, 0x00, 0x00}
	if !bytes.Contains(got, want) {
		t.Fatalf("foreign Monster Riding base stat missing nOption=1902000,rOption=1004; got % x", got)
	}
}

// TestCTSMonsterRidingV83MaskAndNoDoubleEncode verifies the v83 mount GIVE_BUFF
// layout: the TwoState/RideVehicle mask bit lands in mask dword[1] (wire bytes 4-7)
// where the v83 client reads it (registry shift 85 -> uint32(H&0xFFFFFFFF); client
// flag 1<<(i+82) AND'd against wire bytes 4-7), and the stat is encoded only as a
// base stat (no truncated per-stat block). Regression for the mount not rendering:
// the real bug was the per-stat double-encode, not the mask placement.
func TestCTSMonsterRidingV83MaskAndNoDoubleEncode(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeMonsterRiding), 1004, 1902000, 1, time.Now().Add(time.Hour))

	got := input.Encode(nil, ctx)(nil)

	// Mask dword[1] (bytes 4-7) = 0x00200000 -> LE 00 00 20 00: RideVehicle alone,
	// not the whole two-state group.
	if !bytes.Equal(got[4:8], []byte{0x00, 0x00, 0x20, 0x00}) {
		t.Fatalf("mask dword[1] should carry RideVehicle 0x00200000 alone; got % x", got[4:8])
	}
	// Mask dword[2] (bytes 8-11) must be empty for a lone MonsterRiding stat.
	if !bytes.Equal(got[8:12], []byte{0, 0, 0, 0}) {
		t.Fatalf("mask dword[2] should be empty; got % x", got[8:12])
	}
	// No truncated per-stat block: byte 16+ should be the 2 leading bytes (00 00),
	// not the old int16(1902000)=0x05B0 per-stat value.
	if got[16] != 0 || got[17] != 0 {
		t.Fatalf("expected 2 leading bytes (00 00) after mask, not a per-stat block; got % x", got[16:20])
	}
	// The RideVehicle base stat still carries nOption=1902000, rOption=1004.
	want := []byte{0xb0, 0x05, 0x1d, 0x00, 0xec, 0x03, 0x00, 0x00}
	if !bytes.Contains(got, want) {
		t.Fatalf("RideVehicle base stat (1902000,1004) missing; got % x", got)
	}
}

// TestCTSHomingBeaconPre95PopulatedBlock pins the populated GuidedBullet block
// for the classic 7-member two-state group. The block is
// nOption=mobId | rOption=skillId | 5-byte time | dwMobId=mobId — 17 bytes,
// same size as the empty block, so total packet length is unchanged and the
// two-state mask bits (always set pre-95) are unchanged.
//
// Every in-scope version below GMS 95 with the classic 7-member/17-byte
// GuidedBullet block is listed (PRD §2.1). GMS v61 is deliberately absent —
// its IDA-verified 6-member group uses a narrower 16-byte block (no 5-byte
// bool-prefixed time field) and is covered separately by
// TestCTSHomingBeaconV61PopulatedBlock (task-167). gms_12/gms_48 are also
// absent: they take the legacyGmsMask path and have no base-stat trailer at
// all — covered by the negative test below instead.
func TestCTSHomingBeaconPre95PopulatedBlock(t *testing.T) {
	pre95 := []struct {
		name   string
		region string
		major  uint16
	}{
		{"GMS v72", "GMS", 72},
		{"GMS v79", "GMS", 79},
		{"GMS v83", "GMS", 83},
		{"GMS v84", "GMS", 84},
		{"GMS v87", "GMS", 87},
		{"GMS v92", "GMS", 92},
		{"JMS v185", "JMS", 185},
	}
	for _, v := range pre95 {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			tn, _ := tenant.Create([16]byte{}, v.region, v.major, 1)
			input := NewCharacterTemporaryStat()
			// mobId 1000001 (0x000F4241), skill 5211006 (0x004F837E).
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5211006, 1000001, 1, time.Time{})

			got := input.Encode(nil, ctx)(nil)

			// nOption=1000001 then rOption=5211006 as consecutive LE int32s.
			head := []byte{0x41, 0x42, 0x0F, 0x00, 0x7E, 0x83, 0x4F, 0x00}
			idx := bytes.Index(got, head)
			if idx < 0 {
				t.Fatalf("populated GuidedBullet head (nOption=1000001,rOption=5211006) missing; got % x", got)
			}
			// dwMobId sits after the 5-byte DecodeTime: head(8) + time(5) = offset 13.
			mob := got[idx+13 : idx+17]
			if !bytes.Equal(mob, []byte{0x41, 0x42, 0x0F, 0x00}) {
				t.Fatalf("dwMobId: got % x want 41 42 0f 00", mob)
			}
		})
	}
}

// An empty CTS emits no two-state trailer at all. Base bits and base blocks are
// both presence-gated (task-190), so a CTS holding nothing claims nothing: 16
// mask + the 2 leading defense bytes, on every version that takes the
// non-legacy path.
//
// This test used to assert a fixed per-version trailer — 88 bytes on GMS v61's
// 6-member group, 110 on the classic 7-member group — because absent members
// were emitted as empty placeholder blocks. Those constants now live on
// TestCTSTwoStateGroupShape below, which exercises them by populating the group
// rather than by padding it.
func TestCTSAbsentTwoStateStatsEmitNoTrailer(t *testing.T) {
	for _, v := range []struct {
		name   string
		region string
		major  uint16
	}{
		{"GMS v61", "GMS", 61},
		{"GMS v72", "GMS", 72},
		{"GMS v79", "GMS", 79},
		{"GMS v83", "GMS", 83},
		{"GMS v84", "GMS", 84},
		{"GMS v87", "GMS", 87},
		{"GMS v92", "GMS", 92},
		{"JMS v185", "JMS", 185},
	} {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			input := NewCharacterTemporaryStat()

			got := input.Encode(nil, ctx)(nil)

			if len(got) != 16+2 {
				t.Fatalf("empty %s CTS length: got %d want %d (mask + defense bytes, no trailer)", v.name, len(got), 16+2)
			}
			if !bytes.Equal(got[0:16], make([]byte, 16)) {
				t.Fatalf("empty %s CTS mask: got % x want all zero", v.name, got[0:16])
			}
		})
	}
}

// TestCTSTwoStateGroupShape is the falsifier for two-state group membership and
// block sizes (PRD gap 6), and the home of the constants the old empty-trailer
// assertion carried. Populating every member of a version's group makes the
// whole trailer appear, so its total length still encodes the group shape.
//
// GMS v61 is IDA-verified (task-167) as a 6-member group (no Undead) with a
// 12-byte base block (narrowTimeField, no leading bool-prefixed time byte):
// 14+14+14+12+18+16 = 88. Every other pre-95 version uses the classic 7-member
// group at 15+15+15+13+20+17+15 = 110.
//
// If one of these fails, do NOT adjust the constant to make it pass — that
// version's group differs and must be established from IDA evidence first.
func TestCTSTwoStateGroupShape(t *testing.T) {
	for _, v := range []struct {
		name       string
		region     string
		major      uint16
		trailerLen int
	}{
		{"GMS v61", "GMS", 61, 88},
		{"GMS v72", "GMS", 72, 110},
		{"GMS v79", "GMS", 79, 110},
		{"GMS v83", "GMS", 83, 110},
		{"GMS v84", "GMS", 84, 110},
		{"GMS v87", "GMS", 87, 110},
		{"GMS v92", "GMS", 92, 110},
		{"JMS v185", "JMS", 185, 110},
	} {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			tn, _ := tenant.Create([16]byte{}, v.region, v.major, 1)
			input := NewCharacterTemporaryStat()
			for _, bs := range twoStateBaseStats(tn) {
				input.AddStat(nil)(tn)(string(bs.name), 1, 1, 1, time.Now().Add(time.Minute))
			}

			got := input.Encode(nil, ctx)(nil)

			want := 16 + 2 + v.trailerLen
			if len(got) != want {
				t.Fatalf("fully-populated %s two-state trailer: got %d want %d", v.name, len(got), want)
			}
		})
	}
}

// TestCTSHomingBeaconV61PopulatedBlock pins the populated GuidedBullet block
// for GMS v61's IDA-verified 6-member two-state group (task-167). The block
// is nOption=mobId | rOption=skillId | plain 4-byte field (narrowTimeField,
// no bool prefix) | dwMobId=mobId — 16 bytes total (vs 17 on every other
// pre-95 version, whose base uses the 5-byte bool-prefixed time field
// instead). The beacon is the only stat held, so it is the only block on the
// wire: 16 mask + 2 defense + 16.
func TestCTSHomingBeaconV61PopulatedBlock(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 61, 1)
	input := NewCharacterTemporaryStat()
	// mobId 1000001 (0x000F4241), skill 5211006 (0x004F837E).
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5211006, 1000001, 1, time.Time{})

	got := input.Encode(nil, ctx)(nil)

	// 16 mask + 2 leading + the one 16-byte GuidedBullet block.
	if len(got) != 16+2+16 {
		t.Fatalf("v61 beacon packet length: got %d want %d", len(got), 16+2+16)
	}

	// nOption=1000001 then rOption=5211006 as consecutive LE int32s.
	head := []byte{0x41, 0x42, 0x0F, 0x00, 0x7E, 0x83, 0x4F, 0x00}
	idx := bytes.Index(got, head)
	if idx < 0 {
		t.Fatalf("v61 populated GuidedBullet head (nOption=1000001,rOption=5211006) missing; got % x", got)
	}
	// dwMobId sits after the base's plain 4-byte third field (no 5-byte
	// bool-prefixed time): head(8) + plain field(4) = offset 12.
	mob := got[idx+12 : idx+16]
	if !bytes.Equal(mob, []byte{0x41, 0x42, 0x0F, 0x00}) {
		t.Fatalf("v61 dwMobId: got % x want 41 42 0f 00", mob)
	}
}

// TestCTSHomingBeaconV61RoundTrip guards encode/decode symmetry for v61's
// narrower base/block shapes: whatever the encoder writes, the decoder must
// consume, without desyncing the reader for the remainder of the payload.
func TestCTSHomingBeaconV61RoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 61, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5211006, 1000001, 1, time.Time{})
	output := NewCharacterTemporaryStat()
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
}

// gms_12 / gms_48 take the legacyGmsMask path (Region GMS && major < 61):
// 8-byte mask, no nDefenseAtt/nDefenseState, no base-stat trailer. The client
// never reads the two-state bits (IDA: v48 OnTemporaryStatReset @0x71b054 reads
// DecodeBuffer(&v8, 8) @0x71b06e). A HOMING_BEACON stat must therefore produce
// no trailer at all — the beacon is n/a on these versions (PRD §2.1), and this
// test proves that in code rather than asserting it in prose.
func TestCTSHomingBeaconLegacyVersionsHaveNoTrailer(t *testing.T) {
	for _, major := range []uint16{12, 48} {
		t.Run(fmt.Sprintf("GMS v%d", major), func(t *testing.T) {
			ctx := pt.CreateContext("GMS", major, 1)
			tn, _ := tenant.Create([16]byte{}, "GMS", major, 1)
			input := NewCharacterTemporaryStat()
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5211006, 1000001, 1, time.Time{})

			got := input.Encode(nil, ctx)(nil)

			// 8-byte mask only: the beacon is a base-stat-only member, and the
			// legacy path emits no base-stat blocks.
			if len(got) != 8 {
				t.Fatalf("legacy GMS v%d CTS with beacon: got %d bytes want 8 (mask only); trailer must not be emitted", major, len(got))
			}
		})
	}
}

// TestCTSHomingBeaconV95MaskAndBlock pins the v95 beacon give: bit 127
// (0x80000000 in wire dword[0]) and one populated 17-byte GuidedBullet block,
// with no other two-state bit or block, since the CTS holds nothing else. IDA:
// v95 group @SecondaryStat::SecondaryStat 0x72F190, GuidedBullet
// DecodeForClient 0x727180, mask-gated tail read 0x73DBA0 (design.md §2.4).
func TestCTSHomingBeaconV95MaskAndBlock(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5220011, 1000001, 10, time.Time{})

	got := input.Encode(nil, ctx)(nil)

	// dword[0] = 0x80000000 -> LE bytes 00 00 00 80.
	if !bytes.Equal(got[0:4], []byte{0x00, 0x00, 0x00, 0x80}) {
		t.Fatalf("v95 mask dword[0]: got % x want 00 00 00 80", got[0:4])
	}
	if !bytes.Equal(got[4:16], make([]byte, 12)) {
		t.Fatalf("v95 mask dwords[1..3] should be empty; got % x", got[4:16])
	}
	// 16 mask + 2 leading + 17 GuidedBullet.
	if len(got) != 16+2+17 {
		t.Fatalf("v95 beacon packet length: got %d want %d", len(got), 16+2+17)
	}
	// Populated block: nOption=1000001, rOption=5220011 (0x004FA6AB).
	head := []byte{0x41, 0x42, 0x0F, 0x00, 0xAB, 0xA6, 0x4F, 0x00}
	idx := bytes.Index(got, head)
	if idx < 0 {
		t.Fatalf("v95 populated GuidedBullet head missing; got % x", got)
	}
	if !bytes.Equal(got[idx+13:idx+17], []byte{0x41, 0x42, 0x0F, 0x00}) {
		t.Fatalf("v95 dwMobId: got % x want 41 42 0f 00", got[idx+13:idx+17])
	}
}

// An empty v95 CTS claims nothing and carries no trailer.
func TestCTSEmptyV95ClaimsNothing(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	got := input.Encode(nil, ctx)(nil)
	if len(got) != 16+2 {
		t.Fatalf("empty v95 CTS length: got %d want %d", len(got), 16+2)
	}
	if !bytes.Equal(got[0:16], make([]byte, 16)) {
		t.Fatalf("empty v95 mask: got % x want all zero", got[0:16])
	}
}

// TestCTSPartyBoosterV95Block pins the PartyBooster member: bit 126
// (0x40000000) and a 20-byte block (base 13 + tCurrentTime 5 + usExpireTerm 2 —
// IDA DecodeForClient 0x72C600). PartyBooster has no producer in atlas yet;
// this exercises the verified wire slot only.
func TestCTSPartyBoosterV95Block(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypePartyBooster), 1005017, 20, 20, time.Now().Add(time.Minute))

	got := input.Encode(nil, ctx)(nil)

	if !bytes.Equal(got[0:4], []byte{0x00, 0x00, 0x00, 0x40}) {
		t.Fatalf("v95 mask dword[0] with PartyBooster: got % x want 00 00 00 40", got[0:4])
	}
	if len(got) != 16+2+20 {
		t.Fatalf("v95 PartyBooster packet length: got %d want %d", len(got), 16+2+20)
	}
}

// Decode must mirror the conditional read: beacon- and PartyBooster-bearing
// v95 payloads round-trip without desyncing the reader.
func TestCTSHomingBeaconV95RoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5220011, 1000001, 10, time.Time{})
	output := NewCharacterTemporaryStat()
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
}

func TestCTSPartyBoosterV95RoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypePartyBooster), 1005017, 20, 20, time.Now().Add(time.Minute))
	output := NewCharacterTemporaryStat()
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
}

// Foreign v95 encode must never carry the GuidedBullet block even if a beacon
// stat is (incorrectly) present upstream: HOMING_BEACON is caster-only and the
// remote-reader path is unverified (FR-4.5). The lib guarantees this by CTS
// construction (channel never AddStats it on foreign bodies); this test pins
// that an EMPTY foreign v95 CTS carries no trailer at all.
func TestCTSForeignEmptyV95ClaimsNothing(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	got := input.EncodeForeign(nil, ctx)(nil)
	if len(got) != 16+2 {
		t.Fatalf("empty foreign v95 CTS length: got %d want %d", len(got), 16+2)
	}
}

// The mask must contain ONLY the stats the CTS holds — never the two-state
// group bits, which EncodeMask used to assert unconditionally. On the reset
// path that made ANY cancel clear every two-state stat client-side (v83 reset
// @0xA2071F, v95 @0x9F2AB0 clear every masked stat); on the set path it made
// every buff give claim a mount.
func TestMaskContainsOnlyActiveStats(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			cts := NewCharacterTemporaryStat()
			cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeInvincible), 2301003, 30, 20, time.Now().Add(time.Minute))

			mask := cts.activeMask()
			reg := buildCharacterTemporaryStatRegistry(tn)
			inv := reg.byName[character.TemporaryStatTypeInvincible]
			if mask.And(inv.mask).IsZero() {
				t.Fatal("mask missing the active stat's bit")
			}
			riding := reg.byName[character.TemporaryStatTypeMonsterRiding]
			if !mask.And(riding.mask).IsZero() {
				t.Fatal("mask must not contain inactive two-state bits")
			}
		})
	}
}

func TestMaskEmptyForEmptyCTS(t *testing.T) {
	cts := NewCharacterTemporaryStat()
	if !cts.activeMask().IsZero() {
		t.Fatal("empty CTS must produce an empty mask")
	}
}

// Movement filter membership per version (IDA: v83 sub_77DC78, v95
// SecondaryStat::IsMovementAffectingStat @0x7208C0, v61/v72/v79/v84/v87/v92/
// JMS per docs/tasks/task-167-homing-beacon-bullseye/evidence/movement-filter.md).
//
// JMS's filter is a wholly different set from every GMS version (evidence:
// movement-filter.md JMS section) — only Stun, GhostMorph, and MonsterRiding
// overlap the GMS v83 list, and JMS's own filter DOES include Invincible
// (unlike every GMS version, where Invincible is never movement-affecting).
// So JMS gets its own in/out lists rather than the shared GMS ones.
func TestMovementAffectingMaskMembership(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			reg := buildCharacterTemporaryStatRegistry(tn)
			mv := MovementAffectingMask(tn)

			var in, out []character.TemporaryStatType
			if tn.Region() == "JMS" {
				in = []character.TemporaryStatType{
					character.TemporaryStatTypeInvincible,
					character.TemporaryStatTypeSoulArrow,
					character.TemporaryStatTypeStun,
					character.TemporaryStatTypeMesoUpByItem,
					character.TemporaryStatTypeGhostMorph,
					character.TemporaryStatTypeWindBreakerFinal,
					character.TemporaryStatTypeElementalReset,
					character.TemporaryStatTypeEventRate,
					character.TemporaryStatTypeBodyPressure,
					character.TemporaryStatTypeSoulStone,
					character.TemporaryStatTypeSwallowDefense,
					character.TemporaryStatTypeMonsterRiding,
				}
				out = []character.TemporaryStatType{
					character.TemporaryStatTypeHomingBeacon,
					character.TemporaryStatTypeSpeed,
					character.TemporaryStatTypeJump,
					character.TemporaryStatTypeWeaken,
					character.TemporaryStatTypeSlow,
					character.TemporaryStatTypeMorph,
					character.TemporaryStatTypeMapleWarrior,
					character.TemporaryStatTypeSeduce,
					character.TemporaryStatTypeDashSpeed,
					character.TemporaryStatTypeDashJump,
				}
			} else {
				in = []character.TemporaryStatType{
					character.TemporaryStatTypeSpeed,
					character.TemporaryStatTypeJump,
					character.TemporaryStatTypeStun,
					character.TemporaryStatTypeWeaken,
					character.TemporaryStatTypeSlow,
					character.TemporaryStatTypeMorph,
					character.TemporaryStatTypeGhostMorph,
					character.TemporaryStatTypeMapleWarrior,
					character.TemporaryStatTypeSeduce,
					character.TemporaryStatTypeMonsterRiding,
					character.TemporaryStatTypeDashSpeed,
					character.TemporaryStatTypeDashJump,
				}
				out = []character.TemporaryStatType{
					character.TemporaryStatTypeHomingBeacon,
					character.TemporaryStatTypeInvincible,
				}
				if tn.Region() == "GMS" && tn.MajorVersion() >= 95 {
					in = append(in,
						character.TemporaryStatTypeFlying,
						character.TemporaryStatTypeFrozen,
						character.TemporaryStatTypeYellowAura,
					)
				}
				if tn.Region() == "GMS" && tn.MajorVersion() == 92 {
					in = append(in,
						character.TemporaryStatTypeFlying,
						character.TemporaryStatTypeFrozen,
					)
				}
			}

			for _, n := range in {
				st, ok := reg.byName[n]
				if !ok {
					continue // stat not enumerated on this version
				}
				if mv.And(st.mask).IsZero() {
					t.Errorf("%s should be movement-affecting on %s", n, v.Name)
				}
			}
			for _, n := range out {
				st, ok := reg.byName[n]
				if !ok {
					continue
				}
				if !mv.And(st.mask).IsZero() {
					t.Errorf("%s should NOT be movement-affecting on %s", n, v.Name)
				}
			}
		})
	}
}

// A no-expiry buff carries the zero time (buff.NewNoExpiryBuff). No client has a
// no-expiry concept, so the encoder must turn that into a saturated duration.
// Encoding it arithmetically instead underflows: year 1 to now overruns an int64
// nanosecond Duration, and the int32 truncation lands on a negative the client
// treats as already-expired — GM hide would flicker off the instant it landed.
func TestNoExpiryStatEncodesSaturatedDuration(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := NewCharacterTemporaryStat()
	// DARK_SIGHT (GM hide) is a value stat, so it carries a per-stat expiry —
	// unlike MONSTER_RIDING and HOMING_BEACON, which are base stats and never
	// reach this field.
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeDarkSight), 9101004, 1, 1, time.Time{})

	got := cts.Encode(nil, ctx)(nil)

	// mask(16) + value int16 + sourceId int32 + expiry int32 -> expiry is the
	// last 4 bytes before the 2 trailing defense bytes.
	expiry := int32(binary.LittleEndian.Uint32(got[len(got)-6 : len(got)-2]))
	if expiry != math.MaxInt32 {
		t.Fatalf("no-expiry DARK_SIGHT wire duration: got %d want MaxInt32 %d", expiry, int32(math.MaxInt32))
	}
}

func TestLegacyDurationUnitsNoExpirySaturates(t *testing.T) {
	if got := legacyDurationUnits(time.Time{}); got != math.MaxInt16 {
		t.Fatalf("no-expiry legacy duration: got %d want MaxInt16 %d", got, int16(math.MaxInt16))
	}
	if got := legacyDurationUnits(time.Now().Add(-time.Minute)); got != 0 {
		t.Fatalf("already-expired legacy duration: got %d want 0", got)
	}
}

// --- task-195 / issue #1196: foreign (observer) disease blocks ---------------

// TestCTSForeignDiseaseCarriesMobSkillKey pins the foreign per-stat block for a
// mob-applied disease the client CAN render remotely. The v83 client reads one
// Decode4 into the stat's REASON field, and CUser::UpdateAffectedSkillList
// (@0x93e344) hands that reason to CUser::ShowAffectedSkillAni (@0x932da6),
// which splits it as mobSkillId | (level << 16) and loads
// MobSkill[id].level[lv].affected. Atlas used to write the disease amount here,
// so the lookup resolved to nothing and observers saw no debuff at all.
func TestCTSForeignDiseaseCarriesMobSkillKey(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := NewCharacterTemporaryStat()
	// Mob skill 123 (Stun) level 3, amount 100.
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeStun), 123, 100, 3, time.Now().Add(15*time.Second))

	got := cts.EncodeForeign(nil, ctx)(nil)

	// mask(16) + Short(mobSkillId) + Short(level) + nDefenseAtt + nDefenseState.
	if len(got) != 16+4+2 {
		t.Fatalf("foreign STUN length: got %d want %d", len(got), 16+4+2)
	}
	want := []byte{0x7b, 0x00, 0x03, 0x00} // 123, 3
	if !bytes.Equal(got[16:20], want) {
		t.Fatalf("foreign STUN reason: got % x want % x", got[16:20], want)
	}
	// The same 32-bit composite the client's Decode4 sees.
	if key := binary.LittleEndian.Uint32(got[16:20]); key != 123|(3<<16) {
		t.Fatalf("foreign STUN reason as Decode4: got %#x want %#x", key, 123|(3<<16))
	}
}

// TestCTSForeignPoisonCarriesValueThenMobSkillKey pins POISON, the one disease
// whose foreign block carries a value: the client tests CTS_Poison twice in a
// row, reading Decode2 (nPoison, per-tick damage) then Decode4 (rPoison, the
// mob-skill key).
func TestCTSForeignPoisonCarriesValueThenMobSkillKey(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := NewCharacterTemporaryStat()
	// Mob skill 125 (Poison) level 4, 60 damage per tick.
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypePoison), 125, 60, 4, time.Now().Add(15*time.Second))

	got := cts.EncodeForeign(nil, ctx)(nil)

	if len(got) != 16+6+2 {
		t.Fatalf("foreign POISON length: got %d want %d", len(got), 16+6+2)
	}
	want := []byte{0x3c, 0x00, 0x7d, 0x00, 0x04, 0x00} // 60, 125, 4
	if !bytes.Equal(got[16:22], want) {
		t.Fatalf("foreign POISON block: got % x want % x", got[16:22], want)
	}
}

// TestCTSForeignSlowIsMaskOnly pins that SLOW stays mask-only on the foreign
// path. No supported client's SecondaryStat::DecodeForRemote has a CTS_Slow
// branch (v83 xref on 0xbeffc0, v95 xref on 0xc6c9a0, block enumeration
// elsewhere), so emitting a mob-skill key here would be 4 bytes the client
// consumes as nDefenseAtt/nDefenseState.
func TestCTSForeignSlowIsMaskOnly(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeSlow), 126, 80, 2, time.Now().Add(15*time.Second))

	got := cts.EncodeForeign(nil, ctx)(nil)

	if len(got) != 16+2 {
		t.Fatalf("foreign SLOW length: got %d want %d (mask + defense bytes only)", len(got), 16+2)
	}
	// Bit 32 -> mask dword[2] (wire bytes 8-11) = 0x00000001.
	if !bytes.Equal(got[8:12], []byte{0x01, 0x00, 0x00, 0x00}) {
		t.Fatalf("foreign SLOW mask bit: got % x want 01 00 00 00", got[8:12])
	}
}

// TestCTSForeignOrderMatchesClientReadOrder pins that foreign blocks are written
// in SecondaryStat::DecodeForRemote's order, not the registry's shift order. The
// remote decoder reads DARKNESS (shift 20) before SEAL (shift 19) — v83 code
// positions 0x788289 and 0x7882d2 — so a shift sort swapped the two reasons and
// each disease rendered the other's animation.
func TestCTSForeignOrderMatchesClientReadOrder(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeSeal), 120, 1, 1, time.Now().Add(time.Minute))
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeDarkness), 121, 1, 2, time.Now().Add(time.Minute))

	got := cts.EncodeForeign(nil, ctx)(nil)

	if len(got) != 16+4+4+2 {
		t.Fatalf("foreign SEAL+DARKNESS length: got %d want %d", len(got), 16+4+4+2)
	}
	darkness := []byte{0x79, 0x00, 0x02, 0x00} // mob skill 121 level 2
	seal := []byte{0x78, 0x00, 0x01, 0x00}     // mob skill 120 level 1
	if !bytes.Equal(got[16:20], darkness) {
		t.Fatalf("first foreign block should be DARKNESS: got % x want % x", got[16:20], darkness)
	}
	if !bytes.Equal(got[20:24], seal) {
		t.Fatalf("second foreign block should be SEAL: got % x want % x", got[20:24], seal)
	}
}

// TestCTSForeignMultiDiseaseRoundTrip exercises encode/decode symmetry for a
// multi-disease foreign body on every supported version — the ordering fix has
// to move both sides together, or the second block decodes from the first
// block's bytes.
func TestCTSForeignMultiDiseaseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterTemporaryStat()
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeSeal), 120, 1, 1, time.Now().Add(time.Minute))
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeDarkness), 121, 1, 2, time.Now().Add(time.Minute))
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypePoison), 125, 60, 4, time.Now().Add(time.Minute))
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeSpeed), 2001002, 20, 10, time.Now().Add(time.Minute))

			output := NewCharacterTemporaryStat()
			pt.RoundTrip(t, ctx, input.EncodeForeign, output.DecodeForeign, nil)

			for _, c := range []struct {
				name     character.TemporaryStatType
				sourceId int32
				level    byte
			}{
				{character.TemporaryStatTypeSeal, 120, 1},
				{character.TemporaryStatTypeDarkness, 121, 2},
				{character.TemporaryStatTypePoison, 125, 4},
			} {
				sv, ok := output.stats[c.name]
				if !ok {
					t.Fatalf("%s missing after round trip", c.name)
				}
				if sv.SourceId() != c.sourceId || sv.Level() != c.level {
					t.Errorf("%s: got mobSkill %d level %d, want %d/%d", c.name, sv.SourceId(), sv.Level(), c.sourceId, c.level)
				}
			}
			if sv := output.stats[character.TemporaryStatTypePoison]; sv.Value() != 60 {
				t.Errorf("POISON value: got %v, want 60", sv.Value())
			}
		})
	}
}

// TestForeignReadOrderCoversEveryValueCarryingStat is the guard behind
// foreignOrderedTypes' shift-ordered tail: every stat that writes foreign bytes
// must be named in foreignReadOrder, or it encodes at a position the client does
// not read it from. Stats outside the list must be flag-only (zero bytes).
func TestForeignReadOrderCoversEveryValueCarryingStat(t *testing.T) {
	named := make(map[character.TemporaryStatType]bool, len(foreignReadOrder))
	for _, n := range foreignReadOrder {
		named[n] = true
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			reg := buildCharacterTemporaryStatRegistry(tn)
			for _, st := range reg.inOrder {
				if named[st.name] || baseStatNames[st.name] {
					continue
				}
				w := response.NewWriter(nil)
				st.foreignValueWriter(CharacterTemporaryStatValue{statType: st, value: 1, sourceId: 1, level: 1})(w)
				if n := len(w.Bytes()); n != 0 {
					t.Errorf("%s writes %d foreign bytes but is missing from foreignReadOrder", st.name, n)
				}
			}
		})
	}
}

// TestForeignReadOrderNamesOnlyRealStats keeps foreignReadOrder honest: a
// mistyped or removed constant would silently fall through to the shift-ordered
// tail. Every entry must exist in at least one supported version's registry.
func TestForeignReadOrderNamesOnlyRealStats(t *testing.T) {
	known := make(map[character.TemporaryStatType]bool)
	for _, v := range pt.Variants {
		tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
		for name := range buildCharacterTemporaryStatRegistry(tn).byName {
			known[name] = true
		}
	}
	for _, n := range foreignReadOrder {
		if !known[n] {
			t.Errorf("foreignReadOrder names %q, which no supported version's registry has", n)
		}
	}
}

// TestCTSEnergyChargePre95PopulatedBlock pins the ENERGY_CHARGE base block's
// two leading int32s. The client reads nOption as the energy-bar reading:
// GMS v83 IDB sub_7F9BAD computes fill = this[364] / this[365] * bar width,
// where this[364] is the first field of the received two-state entry
// (design.md §1.1). A zeroed nOption renders an empty bar no matter what the
// buff actually holds.
func TestCTSEnergyChargePre95PopulatedBlock(t *testing.T) {
	pre95 := []struct {
		name   string
		region string
		major  uint16
	}{
		{"GMS v72", "GMS", 72},
		{"GMS v79", "GMS", 79},
		{"GMS v83", "GMS", 83},
		{"GMS v84", "GMS", 84},
		{"GMS v87", "GMS", 87},
		{"GMS v92", "GMS", 92},
		{"GMS v95", "GMS", 95},
		{"JMS v185", "JMS", 185},
	}
	for _, v := range pre95 {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			tn, _ := tenant.Create([16]byte{}, v.region, v.major, 1)
			input := NewCharacterTemporaryStat()
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeEnergyCharge), 5110001, 4998, 1, time.Time{})

			got := input.Encode(nil, ctx)(nil)

			// 16 mask + 2 leading defense bytes + one 15-byte dynamic base block.
			if len(got) != 16+2+15 {
				t.Fatalf("energy charge packet length: got %d want %d", len(got), 16+2+15)
			}
			// nOption=4998 then rOption=5110001 as consecutive LE int32s.
			head := []byte{0x86, 0x13, 0x00, 0x00, 0xF1, 0xF8, 0x4D, 0x00}
			if !bytes.Contains(got, head) {
				t.Fatalf("populated ENERGY_CHARGE head (nOption=4998,rOption=5110001) missing; got % x", got)
			}
		})
	}
}

// TestCTSEnergyChargeV61PopulatedBlock covers GMS v61's narrower base block
// (14 bytes: the third field is a bare Decode4, not the bool-prefixed
// 5-byte time pair). Only the block width differs; the two leading int32s
// are in the same place.
func TestCTSEnergyChargeV61PopulatedBlock(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 61, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeEnergyCharge), 5110001, 4998, 1, time.Time{})

	got := input.Encode(nil, ctx)(nil)

	if len(got) != 16+2+14 {
		t.Fatalf("v61 energy charge packet length: got %d want %d", len(got), 16+2+14)
	}
	head := []byte{0x86, 0x13, 0x00, 0x00, 0xF1, 0xF8, 0x4D, 0x00}
	if !bytes.Contains(got, head) {
		t.Fatalf("v61 populated ENERGY_CHARGE head missing; got % x", got)
	}
}

// TestCTSEnergyChargeRoundTrip guards encode/decode symmetry: the decoder is
// shape-only, so a populated block must still be consumed byte-for-byte.
func TestCTSEnergyChargeRoundTrip(t *testing.T) {
	for _, v := range []struct {
		region string
		major  uint16
	}{{"GMS", 61}, {"GMS", 83}, {"GMS", 95}, {"JMS", 185}} {
		ctx := pt.CreateContext(v.region, v.major, 1)
		tn, _ := tenant.Create([16]byte{}, v.region, v.major, 1)
		input := NewCharacterTemporaryStat()
		input.AddStat(nil)(tn)(string(character.TemporaryStatTypeEnergyCharge), 5110001, 4998, 1, time.Time{})
		output := NewCharacterTemporaryStat()
		pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	}
}

// TestCTSDashSpeedStaysZeroed is the negative half of the ENERGY_CHARGE fix:
// the remaining twoStateDynamic member (DASH_SPEED; DASH_JUMP shares the same
// treatment) keeps the zeroed block, because no evidence was gathered for
// what its client reads and this task must not make an unverified wire
// change to a verified cell (design.md §1.1). UNDEAD moved to a populated
// block — see TestCTSUndeadPopulatedBlock — on the IDA evidence in
// bug-zombify-no-visible-effect.md (CUser::UpdateAffectedSkillList
// @0x93e344).
func TestCTSDashSpeedStaysZeroed(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeDashSpeed), 5110001, 4998, 1, time.Time{})

	got := input.Encode(nil, ctx)(nil)

	block := got[18:]
	for i := 0; i < 8; i++ {
		if block[i] != 0x00 {
			t.Fatalf("DASH_SPEED base block must stay zeroed; got % x", block)
		}
	}
}

// TestCTSUndeadPopulatedBlock is the UNDEAD half of the zombify fix
// (bug-zombify-no-visible-effect.md). CUser::UpdateAffectedSkillList
// @0x93e344 keys the client's zombify animation off this block's rOption,
// via sub_672293 (@0x672293, this+4); TemporaryStatBase<long>::
// DecodeForClient @0x793ef2 confirms rOption is the second int32 in wire
// order. rOption carries the mob-skill composite id | (level << 16), the
// same convention MobSkillReasonForeignValueWriter (:306-329) writes for
// every other disease -- not the disease amount.
func TestCTSUndeadPopulatedBlock(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	input := NewCharacterTemporaryStat()
	// Mob skill 133 (Undead) level 5, amount=1.
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeUndead), 133, 1, 5, time.Time{})

	got := input.Encode(nil, ctx)(nil)

	// nOption=1 then rOption=(133 | 5<<16)=327813 as consecutive LE int32s.
	head := []byte{0x01, 0x00, 0x00, 0x00, 0x85, 0x00, 0x05, 0x00}
	if !bytes.Contains(got, head) {
		t.Fatalf("populated UNDEAD head (nOption=1,rOption=327813) missing; got % x", got)
	}
}

// TestCTSUndeadRoundTrip guards encode/decode symmetry: the decoder is
// shape-only, so a populated block must still be consumed byte-for-byte.
func TestCTSUndeadRoundTrip(t *testing.T) {
	for _, v := range []struct {
		region string
		major  uint16
	}{{"GMS", 83}, {"GMS", 95}, {"JMS", 185}} {
		ctx := pt.CreateContext(v.region, v.major, 1)
		tn, _ := tenant.Create([16]byte{}, v.region, v.major, 1)
		input := NewCharacterTemporaryStat()
		input.AddStat(nil)(tn)(string(character.TemporaryStatTypeUndead), 133, 1, 5, time.Time{})
		output := NewCharacterTemporaryStat()
		pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	}
}

// TestCTSServerOnlyStatsSkippedSilently proves PUPPET/SUMMON never reach the
// wire and never log at ERROR (task-164 FR-1/FR-3, acceptance (a)). Both the
// self and foreign encodes of a CTS holding only server-only stats must be
// byte-identical to a freshly-constructed empty CTS, on EVERY supported tenant
// version (FR-7/FR-7.1 — all seven registry classes, including the legacy
// pre-v61 8-byte-mask class). The two AddStat calls must each log exactly one
// DEBUG entry (skip trace) and nothing at ERROR.
//
// Zero expiry (time.Time{}) is used throughout: it saturates to a constant
// duration on both the modern and legacy writers, so byte comparisons are
// deterministic and need no offset arithmetic (design §5.1).
func TestCTSServerOnlyStatsSkippedSilently(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			l, hook := testlog.NewNullLogger()
			l.SetLevel(logrus.DebugLevel)

			// sourceId/amount/level are arbitrary; only wire disposition matters.
			input := NewCharacterTemporaryStat()
			input.AddStat(l)(tn)(string(character.TemporaryStatTypePuppet), 1, 1, 1, time.Time{})
			input.AddStat(l)(tn)(string(character.TemporaryStatTypeSummon), 2, 1, 1, time.Time{})

			for _, e := range hook.AllEntries() {
				if e.Level <= logrus.ErrorLevel {
					t.Errorf("server-only stat add logged at %s: %q", e.Level, e.Message)
				}
			}
			if got := len(hook.AllEntries()); got != 2 {
				t.Errorf("expected exactly 2 DEBUG skip entries, got %d", got)
			}
			for _, e := range hook.AllEntries() {
				if e.Level != logrus.DebugLevel {
					t.Errorf("skip entry logged at %s, want DEBUG: %q", e.Level, e.Message)
				}
			}

			empty := NewCharacterTemporaryStat()
			if got, want := input.Encode(nil, ctx)(nil), empty.Encode(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("Encode with server-only stats differs from empty CTS:\ngot  % x\nwant % x", got, want)
			}
			if got, want := input.EncodeForeign(nil, ctx)(nil), empty.EncodeForeign(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("EncodeForeign with server-only stats differs from empty CTS:\ngot  % x\nwant % x", got, want)
			}
		})
	}
}

// TestCTSAddStatUnknownNameStillErrors pins the existing behavior for
// genuinely unregistered stat names (task-164 acceptance (d)): the stat is
// dropped AND the ERROR log fires. Guards against the server-only skip
// accidentally widening into a general silent-drop. Looped over every
// supported version because the error path must NOT become version-dependent.
func TestCTSAddStatUnknownNameStillErrors(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			l, hook := testlog.NewNullLogger()

			input := NewCharacterTemporaryStat()
			input.AddStat(l)(tn)("BOGUS", 1, 1, 1, time.Time{})

			errorEntries := 0
			for _, e := range hook.AllEntries() {
				if e.Level == logrus.ErrorLevel {
					errorEntries++
					if e.Message != "Attempting to add buff [BOGUS], but cannot find it." {
						t.Errorf("unexpected error message: %q", e.Message)
					}
				}
			}
			if errorEntries != 1 {
				t.Errorf("expected exactly 1 ERROR entry for unknown stat, got %d", errorEntries)
			}

			empty := NewCharacterTemporaryStat()
			if got, want := input.Encode(nil, ctx)(nil), empty.Encode(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("unknown stat leaked into encode:\ngot  % x\nwant % x", got, want)
			}
		})
	}
}

// TestCTSMixedBuffServerOnlyByteInvariance proves a buff carrying both a wire
// stat and server-only stats encodes byte-identically to the same buff without
// the server-only stats (task-164 acceptance (b)), on EVERY supported tenant
// version.
//
// Determinism: the self Encode path writes each per-stat duration as a function
// of expiresAt evaluated at encode time, which is not stable across two Encode
// calls. Passing the zero time saturates that field to a constant on both the
// modern and legacy writers (pinned by TestNoExpiryStatEncodesSaturatedDuration
// and TestLegacyDurationUnitsNoExpirySaturates), so the FULL byte slices compare
// equal with no offset arithmetic — which also keeps this test correct on the
// legacy pre-v61 class, whose mask is 8 bytes rather than 16 (design §5.1).
//
// Booster is the wire-stat probe because it is registered unconditionally
// (shift 11, before any version gate), so it exists in every registry class and
// sits inside the legacy mask's bits 0-46.
func TestCTSMixedBuffServerOnlyByteInvariance(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()

			// Booster + the two server-only stats. sourceId/amount arbitrary;
			// only wire disposition is under test.
			mixed := NewCharacterTemporaryStat()
			mixed.AddStat(l)(tn)(string(character.TemporaryStatTypeBooster), 1001, -2, 1, time.Time{})
			mixed.AddStat(l)(tn)(string(character.TemporaryStatTypePuppet), 1002, 1, 1, time.Time{})
			mixed.AddStat(l)(tn)(string(character.TemporaryStatTypeSummon), 1003, 1, 1, time.Time{})

			plain := NewCharacterTemporaryStat()
			plain.AddStat(l)(tn)(string(character.TemporaryStatTypeBooster), 1001, -2, 1, time.Time{})

			if got, want := mixed.Encode(nil, ctx)(nil), plain.Encode(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("Encode differs:\ngot  % x\nwant % x", got, want)
			}
			if got, want := mixed.EncodeForeign(nil, ctx)(nil), plain.EncodeForeign(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("EncodeForeign differs:\ngot  % x\nwant % x", got, want)
			}
		})
	}
}

// TestCTSPureServerOnlyBuffEncodesAsEmpty proves a buff whose changes are ALL
// server-only yields exactly the empty-CTS body (task-164 acceptance (c),
// FR-5/FR-6): mask claims nothing, no per-stat blocks, standard trailer. The
// buff writers emit unconditionally, so these bytes are what an emitted
// empty-mask GIVE_BUFF / cancel-reset carries — on every supported version,
// including the legacy class where that mask is 8 zero bytes.
func TestCTSPureServerOnlyBuffEncodesAsEmpty(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()

			pure := NewCharacterTemporaryStat()
			pure.AddStat(l)(tn)(string(character.TemporaryStatTypePuppet), 1, 1, 1, time.Time{})

			empty := NewCharacterTemporaryStat()
			if got, want := pure.Encode(nil, ctx)(nil), empty.Encode(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("pure server-only Encode differs from empty CTS:\ngot  % x\nwant % x", got, want)
			}
			if got, want := pure.EncodeForeign(nil, ctx)(nil), empty.EncodeForeign(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("pure server-only EncodeForeign differs from empty CTS:\ngot  % x\nwant % x", got, want)
			}
		})
	}
}
