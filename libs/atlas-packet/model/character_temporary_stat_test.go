package model

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
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

	// Mask: SLOW plus the always-present TwoState base stat bits
	// (EnergyCharge..Undead). The registry assigns the TwoState group shifts 82-88 on
	// v83 -> all land in the high 64 bits, so uint32(H&0xFFFFFFFF)=0x01FC0000 is written
	// to mask dword[1] (wire bytes 4-7), with RideVehicle at 0x00200000. This matches the
	// v83 client's flag 1<<(i+82) read from wire bytes 4-7 (IDA SecondaryStat::
	// DecodeForLocal @0x781D0E; UINT128 dword array is big-endian, AND'd in wire order).
	// SLOW (shift 32) lands in dword[2] (wire bytes 8-11) at 0x00000001 -> LE 01 00 00 00.
	wantMask := []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xFC, 0x01,
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

	// Mask dword[0] (bytes 0-3) = 0x3C000000 -> LE 00 00 00 3C (RideVehicle bit 0x20000000 set).
	if !bytes.Equal(got[0:4], []byte{0x00, 0x00, 0x00, 0x3C}) {
		t.Fatalf("v95 mask dword[0] should be 0x3C000000 (RideVehicle@125 set); got % x", got[0:4])
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
	// 16 mask + 2 leading + base blocks (EnergyCharge15+DashSpeed15+DashJump15+MonsterRiding13 = 58).
	if len(got) != 16+2+58 {
		t.Fatalf("v95 mount packet length: got %d want %d", len(got), 16+2+58)
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

	// Mask dword[1] (bytes 4-7) = 0x01FC0000 -> LE 00 00 FC 01, includes RideVehicle 0x00200000.
	if !bytes.Equal(got[4:8], []byte{0x00, 0x00, 0xFC, 0x01}) {
		t.Fatalf("mask dword[1] should carry TwoState 0x01FC0000 (RideVehicle bit set); got % x", got[4:8])
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
// Every in-scope version below GMS 95 is listed (PRD §2.1). gms_12/gms_48 are
// deliberately absent: they take the legacyGmsMask path and have no base-stat
// trailer at all — covered by the negative test below instead.
func TestCTSHomingBeaconPre95PopulatedBlock(t *testing.T) {
	pre95 := []struct {
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

// Without an active beacon the encode must stay byte-compatible with today:
// the GuidedBullet slot still emits an empty 17-byte block (nOption=0).
//
// The length assertion runs per version and is the cheap falsifier for group
// membership (PRD gap 6): 110 trailer bytes == the 7-member group holds for
// that version. If one of these fails, do NOT adjust the constant to make it
// pass — that version's group differs and Task 7 must establish its real shape.
func TestCTSHomingBeaconPre95AbsentStaysEmpty(t *testing.T) {
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

			// Empty pre-95 CTS: 16 mask + 2 leading + 7 base blocks
			// (15+15+15+13+20+17+15 = 110).
			if len(got) != 16+2+110 {
				t.Fatalf("empty %s CTS length: got %d want %d", v.name, len(got), 16+2+110)
			}
		})
	}
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
// (0x80000000 in wire dword[0]) joins the 4 always-set two-state bits
// (0x3C000000), and the trailer is the status-quo 58 bytes plus one populated
// 17-byte GuidedBullet block. IDA: v95 group @SecondaryStat::SecondaryStat
// 0x72F190, GuidedBullet DecodeForClient 0x727180, mask-gated tail read
// 0x73DBA0 (design.md §2.4).
func TestCTSHomingBeaconV95MaskAndBlock(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5220011, 1000001, 10, time.Time{})

	got := input.Encode(nil, ctx)(nil)

	// dword[0] = 0x3C000000 | 0x80000000 = 0xBC000000 -> LE bytes 00 00 00 BC.
	if !bytes.Equal(got[0:4], []byte{0x00, 0x00, 0x00, 0xBC}) {
		t.Fatalf("v95 mask dword[0]: got % x want 00 00 00 bc", got[0:4])
	}
	if !bytes.Equal(got[4:16], make([]byte, 12)) {
		t.Fatalf("v95 mask dwords[1..3] should be empty; got % x", got[4:16])
	}
	// 16 mask + 2 leading + 58 status-quo blocks + 17 GuidedBullet.
	if len(got) != 16+2+58+17 {
		t.Fatalf("v95 beacon packet length: got %d want %d", len(got), 16+2+58+17)
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

// Non-beacon v95 traffic must stay byte-identical to the current truncated
// encode (regression safety for every existing v95 buff packet).
// TestCTSMonsterRidingV95MaskAndLayout (above) already pins the mount case;
// this pins the empty case.
func TestCTSEmptyV95StaysStatusQuo(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	got := input.Encode(nil, ctx)(nil)
	if len(got) != 16+2+58 {
		t.Fatalf("empty v95 CTS length: got %d want %d", len(got), 16+2+58)
	}
	if !bytes.Equal(got[0:4], []byte{0x00, 0x00, 0x00, 0x3C}) {
		t.Fatalf("empty v95 mask dword[0]: got % x want 00 00 00 3c", got[0:4])
	}
}

// TestCTSPartyBoosterV95Block pins the conditional PartyBooster member:
// bit 126 (0x40000000) and a 20-byte block (base 13 + tCurrentTime 5 +
// usExpireTerm 2 — IDA DecodeForClient 0x72C600). PartyBooster has no
// producer in atlas yet; this exercises the verified wire slot only.
func TestCTSPartyBoosterV95Block(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypePartyBooster), 1005017, 20, 20, time.Now().Add(time.Minute))

	got := input.Encode(nil, ctx)(nil)

	if !bytes.Equal(got[0:4], []byte{0x00, 0x00, 0x00, 0x7C}) {
		t.Fatalf("v95 mask dword[0] with PartyBooster: got % x want 00 00 00 7c", got[0:4])
	}
	if len(got) != 16+2+58+20 {
		t.Fatalf("v95 PartyBooster packet length: got %d want %d", len(got), 16+2+58+20)
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
// that an EMPTY foreign v95 CTS stays at the status-quo length.
func TestCTSForeignEmptyV95StaysStatusQuo(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	got := input.EncodeForeign(nil, ctx)(nil)
	if len(got) != 16+2+58 {
		t.Fatalf("empty foreign v95 CTS length: got %d want %d", len(got), 16+2+58)
	}
}
