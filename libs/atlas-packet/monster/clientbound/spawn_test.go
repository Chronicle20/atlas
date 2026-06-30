package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterSpawn version=gms_v83 ida=0x67945a
// packet-audit:verify packet=monster/clientbound/MonsterSpawn version=gms_v87 ida=0x6b4fa6
// packet-audit:verify packet=monster/clientbound/MonsterSpawn version=gms_v95 ida=0x6589e0
// packet-audit:verify packet=monster/clientbound/MonsterSpawn version=jms_v185 ida=0x6f885c
// packet-audit:verify packet=monster/clientbound/MonsterSpawn version=gms_v84 ida=0x68fff0
func TestMonsterSpawnControlled(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	input := NewMonsterSpawn(5001, true, 100100, m)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestMonsterSpawnUncontrolled(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	input := NewMonsterSpawn(5001, false, 100100, m)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMonsterSpawnBytesV79 pins the exact wire bytes against the v79 client
// read order in CMobPool::OnMobEnterField @0x646e33 (GMS_v79_1_DEVM.exe, port
// 13340):
//
//	Decode4 @0x646e55 — uniqueId
//	Decode1 @0x646e62 — control byte (the "1 vs 5" controller flag)
//	Decode4 @0x646e6f — monsterId (template id)
//	CMob::SetTemporaryStat + CMob::Init @0x646f26/0x646f33 — monster blob
//
// v79 is GMS major 79 (>12, <87): the control byte is present (spawn.go gate
// MajorVersion()>12) and the monster blob omits the v87+ phase field
// (model.go:512 MajorAtLeast(87)). Layout byte-identical to the v83 path; no
// codec change. Empty MonsterTemporaryStat encodes as the bare 16-byte mask.
//
// packet-audit:verify packet=monster/clientbound/MonsterSpawn version=gms_v79 ida=0x646e33
func TestMonsterSpawnBytesV79(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	input := NewMonsterSpawn(5001, true, 100100, m)
	ctx := test.CreateContext("GMS", 79, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001 — Decode4 @0x646e55
		0x01,                   // control byte (controlled) — Decode1 @0x646e62
		0x04, 0x87, 0x01, 0x00, // monsterId 100100 (0x18704) — Decode4 @0x646e6f
		// --- monster blob (model.MonsterModel.Encode, GMS>12 && <87) ---
		0x00, 0x00, 0x00, 0x00, // temp-stat mask H.hi (empty) — model.go:498
		0x00, 0x00, 0x00, 0x00, // mask H.lo
		0x00, 0x00, 0x00, 0x00, // mask L.hi
		0x00, 0x00, 0x00, 0x00, // mask L.lo
		0x64, 0x00, // x 100 — model.go:500
		0xC8, 0x00, // y 200 — model.go:501
		0x05,       // moveAction 5 — model.go:502
		0x00, 0x00, // foothold 0 — model.go:503
		0x2C, 0x01, // homeFoothold 300 — model.go:504
		0xFE,       // appearType -2 (Regen) — model.go:505
		// appearTypeOption omitted (-2 is neither -3 nor >=0) — model.go:506
		0x00,                   // team 0 — model.go:510
		0x00, 0x00, 0x00, 0x00, // effectItemId 0 — model.go:511
		// phase omitted (<87) — model.go:512
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 spawn bytes:\n got % x\nwant % x", got, want)
	}
}
