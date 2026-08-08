package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestMonsterSpawnBytesV48 pins the v48 spawn wire, which is the v61 wire MINUS
// the trailing team byte and effectItemId int.
//
// IDA (GMS_v48_1_DEVM.exe): CMobPool::OnMobEnterField @0x559445 reads
// Decode4(uniqueId) @0x559467, Decode1(control) @0x559474, Decode4(templateId)
// @0x559481, then hands the stat block to sub_5531D5 @0x5531d5 — one Decode4
// mask @0x5531ff, matching legacyMobStatMask (<79) — and the position block to
// CMob::Init (sub_549040 @0x549040). That body reads exactly seven fields:
//
//	Decode2 @0x54908c → x
//	Decode2 @0x54909a → y
//	Decode1 @0x5490ba → moveAction
//	Decode2 @0x5490d6 → foothold
//	Decode2 @0x5490e0 → homeFoothold
//	Decode1 @0x5490ef → appearType
//	Decode4 @0x54910a → appearTypeOption, only when appearType == -3 || >= 0
//
// and then makes no further CInPacket call — the remainder of the function is
// layer/canvas/sound setup. The client's -3 guard is MonsterAppearTypeRevived,
// so the existing conditional matches.
//
// v61 CMob::Init @0x5c2717, v72 @0x6122d4 and v79 @0x6312a7 read nine: the same
// seven plus Decode1 (team) and Decode4 (effectItemId). Atlas gated that pair on
// MajorVersion() > 12, so v48 received five bytes it never reads per mob.
//
// packet-audit:verify packet=monster/clientbound/MonsterSpawn version=gms_v48 ida=0x559445
func TestMonsterSpawnBytesV48(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	input := NewMonsterSpawn(5001, true, 100100, m)
	ctx := test.CreateContext("GMS", 48, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001 — Decode4 @0x559467
		0x01,                   // control byte — Decode1 @0x559474
		0x04, 0x87, 0x01, 0x00, // monsterId 100100 — Decode4 @0x559481
		0x00, 0x00, 0x00, 0x00, // temp-stat mask (empty, legacy 4-byte) — Decode4 @0x5531ff
		0x64, 0x00, // x 100          — Decode2 @0x54908c
		0xC8, 0x00, // y 200          — Decode2 @0x54909a
		0x05,       // moveAction 5   — Decode1 @0x5490ba
		0x00, 0x00, // foothold 0     — Decode2 @0x5490d6
		0x2C, 0x01, // homeFoothold   — Decode2 @0x5490e0
		0xFE, // appearType -2 (Regen) — Decode1 @0x5490ef
		// appearTypeOption omitted: -2 is neither -3 nor >= 0.
		// team + effectItemId omitted: absent until v61.
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v48 spawn bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterSpawnTeamEffectItemBoundary guards the v48/v61 boundary directly:
// v48 stops after appearType, v61 onward appends team + effectItemId.
//
// Limited to v61 and v72 because they are the versions that share v48's stat-mask
// width. legacyMobStatMask switches to the 128-bit mask at v79, which widens the
// stat block by 12 bytes and would swamp the 5-byte tail this test isolates.
func TestMonsterSpawnTeamEffectItemBoundary(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	input := NewMonsterSpawn(5001, true, 100100, m)

	v48 := input.Encode(nil, test.CreateContext("GMS", 48, 1))(nil)
	for _, v := range []uint16{61, 72} {
		got := input.Encode(nil, test.CreateContext("GMS", v, 1))(nil)
		if len(got) != len(v48)+5 {
			t.Errorf("GMS v%d length = %d, want %d (v48 + team + effectItemId)", v, len(got), len(v48)+5)
			continue
		}
		if !bytes.Equal(got[:len(v48)], v48) {
			t.Errorf("GMS v%d diverges from v48 before the tail:\n got % x\nwant % x", v, got[:len(v48)], v48)
		}
		if !bytes.Equal(got[len(v48):], []byte{0x00, 0x00, 0x00, 0x00, 0x00}) {
			t.Errorf("GMS v%d tail = % x, want 00 00 00 00 00", v, got[len(v48):])
		}
	}
}
