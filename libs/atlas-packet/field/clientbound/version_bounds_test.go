package clientbound

import (
	"bytes"
	"testing"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// sampleCharacterData mirrors the fixture used by the roundtrip tests so the
// boundary tests exercise the full SetField/WarpToMap body.
func sampleCharacterData() charpkt.CharacterData {
	return charpkt.CharacterData{
		Stats: charpkt.CharacterStats{
			Id: 1000, Name: "TestChar", Gender: 0, SkinColor: 1,
			Face: 20000, Hair: 30000,
			Level: 50, JobId: 312, Str: 100, Dex: 50, Int: 30, Luk: 20,
			Hp: 5000, MaxHp: 5000, Mp: 3000, MaxMp: 3000,
			Ap: 5, Sp: 3, Exp: 50000, Fame: 10,
			MapId: 100000000, SpawnPoint: 0,
		},
		BuddyCapacity: 20,
		Meso:          100000,
		Inventory: charpkt.InventoryData{
			EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
			EtcCapacity: 24, CashCapacity: 24,
			Timestamp: 94354848000000000,
		},
	}
}

// TestSetFieldVersionBoundary pins the corrected >83 -> >=87 boundary
// (delta §3.1.6): v84/v85/v86 must encode byte-identically to v83 (no
// decode-opt short, no logout-gift block). v87 and v95 stay on the v87+ path.
// The model is built once so its embedded timestamp is identical across encodes.
func TestSetFieldVersionBoundary(t *testing.T) {
	m := NewSetField(channel.Id(1), sampleCharacterData())
	encode := func(major uint16) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		return pt.Encode(t, ctx, m.Encode, nil)
	}
	v83 := encode(83)
	for _, major := range []uint16{84, 85, 86} {
		if got := encode(major); !bytes.Equal(got, v83) {
			t.Errorf("SetField v%d encode differs from v83 (len %d vs %d); v84..86 must match v83", major, len(got), len(v83))
		}
	}
	// v87+ path must remain on the v87+ side (longer: +2 decode-opt +16 gift).
	if v87 := encode(87); bytes.Equal(v87, v83) {
		t.Errorf("SetField v87 must stay on the v87+ path, not equal v83")
	}
	if v95 := encode(95); bytes.Equal(v95, v83) {
		t.Errorf("SetField v95 must stay on the v87+ path, not equal v83")
	}
}

// TestWarpToMapVersionBoundary mirrors SetField for the WarpToMap writer,
// which shares the OnSetField parse (delta §3.1.6).
func TestWarpToMapVersionBoundary(t *testing.T) {
	m := NewWarpToMap(channel.Id(1), 100000000, 0, 5000)
	encode := func(major uint16) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		return pt.Encode(t, ctx, m.Encode, nil)
	}
	v83 := encode(83)
	for _, major := range []uint16{84, 85, 86} {
		if got := encode(major); !bytes.Equal(got, v83) {
			t.Errorf("WarpToMap v%d encode differs from v83 (len %d vs %d); v84..86 must match v83", major, len(got), len(v83))
		}
	}
	if v87 := encode(87); bytes.Equal(v87, v83) {
		t.Errorf("WarpToMap v87 must stay on the v87+ path, not equal v83")
	}
}
