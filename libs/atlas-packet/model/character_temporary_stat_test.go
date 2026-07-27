package model

import (
	"bytes"
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
